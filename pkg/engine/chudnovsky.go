package engine

import (
	"context"
	"math/big"
	"sync"
	"sync/atomic"
)

const (
	startDigits     = 10_000
	safetyMargin    = 20
	pipelineDepth   = 4
	writeBufferSize = 4 * 1024 * 1024
)

// ComputePi computes π to `prec` bits of precision using the Chudnovsky
// algorithm.  When workers > 1 a work-stealing atomic counter is used to
// saturate all CPUs.  Each term claim and each worker completion is reported
// via statCh (non-blocking; caller must drain promptly).
// The computation is cancelled cleanly when ctx.Done() fires.
func ComputePi(prec uint, workers int, ctx context.Context, statCh chan<- WorkerStat) *big.Float {
	const (
		A int64 = 13591409
		B int64 = 545140134
		C int64 = -262537412640768000
	)

	terms := int(prec/156) + 3

	// Factorial — allocates a fresh big.Int; goroutine-safe.
	fac := func(n int64) *big.Int {
		if n <= 0 {
			return big.NewInt(1)
		}
		r := big.NewInt(1)
		for i := int64(2); i <= n; i++ {
			r.Mul(r, big.NewInt(i))
		}
		return r
	}

	// computeTerm computes the k-th Chudnovsky term as a big.Float.
	computeTerm := func(k int, cpow *big.Int) *big.Float {
		k64 := int64(k)
		num := fac(6 * k64)
		num.Mul(num, new(big.Int).SetInt64(A+B*k64))
		den := fac(3 * k64)
		kf := fac(k64)
		kf3 := new(big.Int).Mul(kf, kf)
		kf3.Mul(kf3, kf)
		den.Mul(den, kf3)
		den.Mul(den, cpow)
		nf := new(big.Float).SetPrec(prec).SetInt(num)
		df := new(big.Float).SetPrec(prec).SetInt(den)
		return new(big.Float).SetPrec(prec).Quo(nf, df)
	}

	// Pre-compute C^k table sequentially.
	Cbig := new(big.Int).SetInt64(C)
	cpows := make([]*big.Int, terms)
	cpows[0] = big.NewInt(1)
	for k := 1; k < terms; k++ {
		cpows[k] = new(big.Int).Mul(cpows[k-1], Cbig)
	}

	// sendStat sends a WorkerStat non-blocking (best-effort; never stalls compute).
	sendStat := func(stat WorkerStat) {
		if statCh == nil {
			return
		}
		select {
		case statCh <- stat:
		default:
		}
	}

	if workers <= 1 {
		// Single-threaded path.
		sum := new(big.Float).SetPrec(prec)
		for k := 0; k < terms; k++ {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			sendStat(WorkerStat{WorkerID: 0, TermsComputed: int64(k), Active: true})
			sum.Add(sum, computeTerm(k, cpows[k]))
		}
		sendStat(WorkerStat{WorkerID: 0, Active: false})
		return finalise(sum, prec)
	}

	// Multi-threaded: work-stealing via atomic counter.
	var nextTerm atomic.Int64
	partials := make([]*big.Float, workers)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := new(big.Float).SetPrec(prec)
			for {
				// Check cancellation before claiming a term.
				select {
				case <-ctx.Done():
					sendStat(WorkerStat{WorkerID: w, Active: false})
					return
				default:
				}
				k := int(nextTerm.Add(1) - 1)
				if k >= terms {
					break
				}
				sendStat(WorkerStat{WorkerID: w, TermsComputed: int64(k), Active: true})
				local.Add(local, computeTerm(k, cpows[k]))
			}
			sendStat(WorkerStat{WorkerID: w, Active: false})
			partials[w] = local
		}()
	}
	wg.Wait()

	// Check if we were cancelled.
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	sum := new(big.Float).SetPrec(prec)
	for _, p := range partials {
		if p != nil {
			sum.Add(sum, p)
		}
	}
	return finalise(sum, prec)
}

// finalise applies the Chudnovsky constant multiplier: π = 426880·√10005 / sum.
func finalise(sum *big.Float, prec uint) *big.Float {
	sqrt10005 := new(big.Float).SetPrec(prec).SetInt64(10005)
	sqrt10005.Sqrt(sqrt10005)
	coeff := new(big.Float).SetPrec(prec).SetInt64(426880)
	coeff.Mul(coeff, sqrt10005)
	return new(big.Float).SetPrec(prec).Quo(coeff, sum)
}

// piDigitsText returns a decimal string of π for the given bit precision.
func piDigitsText(prec uint, workers int, ctx context.Context, statCh chan<- WorkerStat) string {
	pi := ComputePi(prec, workers, ctx, statCh)
	if pi == nil {
		return ""
	}
	return pi.Text('f', int(prec/4))
}

// afterDot returns everything after the first '.' in s.
func afterDot(s string) string {
	for i, ch := range s {
		if ch == '.' {
			return s[i+1:]
		}
	}
	return s
}
