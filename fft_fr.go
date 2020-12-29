package kate

import "fmt"

func (fs *FFTSettings) simpleFT(vals []Big, valsOffset uint64, valsStride uint64, rootsOfUnity []Big, rootsOfUnityStride uint64, out []Big) {
	l := uint64(len(out))
	var v Big
	var tmp Big
	var last Big
	for i := uint64(0); i < l; i++ {
		jv := &vals[valsOffset]
		r := &rootsOfUnity[0]
		mulModBig(&v, jv, r)
		CopyBigNum(&last, &v)

		for j := uint64(1); j < l; j++ {
			jv := &vals[valsOffset+j*valsStride]
			r := &rootsOfUnity[((i*j)%l)*rootsOfUnityStride]
			mulModBig(&v, jv, r)
			CopyBigNum(&tmp, &last)
			addModBig(&last, &tmp, &v)
		}
		CopyBigNum(&out[i], &last)
	}
}

func (fs *FFTSettings) _fft(vals []Big, valsOffset uint64, valsStride uint64, rootsOfUnity []Big, rootsOfUnityStride uint64, out []Big) {
	if len(out) <= 4 { // if the value count is small, run the unoptimized version instead. // TODO tune threshold.
		fs.simpleFT(vals, valsOffset, valsStride, rootsOfUnity, rootsOfUnityStride, out)
		return
	}

	half := uint64(len(out)) >> 1
	// L will be the left half of out
	fs._fft(vals, valsOffset, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[:half])
	// R will be the right half of out
	fs._fft(vals, valsOffset+valsStride, valsStride<<1, rootsOfUnity, rootsOfUnityStride<<1, out[half:]) // just take even again

	var yTimesRoot Big
	var x, y Big
	for i := uint64(0); i < half; i++ {
		// temporary copies, so that writing to output doesn't conflict with input
		CopyBigNum(&x, &out[i])
		CopyBigNum(&y, &out[i+half])
		root := &rootsOfUnity[i*rootsOfUnityStride]
		mulModBig(&yTimesRoot, &y, root)
		addModBig(&out[i], &x, &yTimesRoot)
		subModBig(&out[i+half], &x, &yTimesRoot)
	}
}

func (fs *FFTSettings) FFT(vals []Big, inv bool) ([]Big, error) {
	n := uint64(len(vals))
	if n > fs.maxWidth {
		return nil, fmt.Errorf("got %d values but only have %d roots of unity", n, fs.maxWidth)
	}
	if !isPowerOfTwo(n) {
		return nil, fmt.Errorf("got %d values but not a power of two", n)
	}
	// We make a copy so we can mutate it during the work.
	valsCopy := make([]Big, n, n) // TODO: maybe optimize this away, and write back to original input array?
	for i := uint64(0); i < n; i++ {
		CopyBigNum(&valsCopy[i], &vals[i])
	}
	if inv {
		var invLen Big
		asBig(&invLen, n)
		invModBig(&invLen, &invLen)
		rootz := fs.reverseRootsOfUnity[:fs.maxWidth]
		stride := fs.maxWidth / n

		out := make([]Big, n, n)
		fs._fft(valsCopy, 0, 1, rootz, stride, out)
		var tmp Big
		for i := 0; i < len(out); i++ {
			mulModBig(&tmp, &out[i], &invLen)
			CopyBigNum(&out[i], &tmp) // TODO: depending on bignum implementation, allow to directly write back to an input
		}
		return out, nil
	} else {
		out := make([]Big, n, n)
		rootz := fs.expandedRootsOfUnity[:fs.maxWidth]
		stride := fs.maxWidth / n
		// Regular FFT
		fs._fft(valsCopy, 0, 1, rootz, stride, out)
		return out, nil
	}
}

// rearrange Big elements in reverse bit order. Supports 2**31 max element count.
func reverseBitOrderBig(values []Big) {
	if len(values) > (1 << 31) {
		panic("list too large")
	}
	var tmp Big
	reverseBitOrder(uint32(len(values)), func(i, j uint32) {
		CopyBigNum(&tmp, &values[i])
		CopyBigNum(&values[i], &values[j])
		CopyBigNum(&values[j], &tmp)
	})
}
