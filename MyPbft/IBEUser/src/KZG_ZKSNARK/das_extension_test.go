package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
	"math/rand"
	"testing"
)

func TestDASFFTExtension(t *testing.T) {
	fs := NewFFTSettings(4)
	half := fs.MaxWidth / 2
	data := make([]kzg_bls.Fr, half, half)
	for i := uint64(0); i < half; i++ {
		kzg_bls.AsFr(&data[i], i)
	}
	debugFrs("even data", data)
	fs.DASFFTExtension(data)
	debugFrs("odd data", data)
	ToFr := func(v string) (out kzg_bls.Fr) {
		kzg_bls.SetFr(&out, v)
		return
	}
	expected := []kzg_bls.Fr{
		ToFr("35517140934261047308355351661356802312031268910108466120070952281657631518077"),
		ToFr("46293835246856164064818777137000049805076132996160294782312647979750015529053"),
		ToFr("16918734240865143167627244020755511206883014059731428924262453949515587703435"),
		ToFr("11473449502290064142245761066479007451139502549599385854846611945573094960557"),
		ToFr("16918734240865143167627244020755511206883014059731428924262453949515587703435"),
		ToFr("46293835246856164064818777137000049805076132996160294782312647979750015529053"),
		ToFr("35517140934261047308355351661356802312031268910108466120070952281657631518077"),
		ToFr("810630354249988693942455328040129251641875520510785782275914432334760276393"),
	}
	for i := range data {
		if got := &data[i]; !kzg_bls.EqualFr(got, &expected[i]) {
			t.Errorf("difference: %d: got: %s  expected: %s", i, kzg_bls.FrStr(got), kzg_bls.FrStr(&expected[i]))
		}
	}
}

func TestParametrizedDASFFTExtension(t *testing.T) {
	testScale := func(seed int64, scale uint8, t *testing.T) {
		fs := NewFFTSettings(scale)
		evenData := make([]kzg_bls.Fr, fs.MaxWidth/2, fs.MaxWidth/2)
		rng := rand.New(rand.NewSource(seed))
		for i := uint64(0); i < fs.MaxWidth/2; i++ {
			kzg_bls.AsFr(&evenData[i], rng.Uint64()) // TODO could be a full random F_r instead of uint64
		}
		debugFrs("input data", evenData)
		// we don't want to modify the original input, and the inner function would modify it in-place, so make a copy.
		oddData := make([]kzg_bls.Fr, fs.MaxWidth/2, fs.MaxWidth/2)
		for i := 0; i < len(oddData); i++ {
			kzg_bls.CopyFr(&oddData[i], &evenData[i])
		}
		fs.DASFFTExtension(oddData)
		debugFrs("output data", oddData)

		// reconstruct data
		data := make([]kzg_bls.Fr, fs.MaxWidth, fs.MaxWidth)
		for i := uint64(0); i < fs.MaxWidth; i += 2 {
			kzg_bls.CopyFr(&data[i], &evenData[i>>1])
			kzg_bls.CopyFr(&data[i+1], &oddData[i>>1])
		}
		debugFrs("reconstructed data", data)
		// get coefficients of reconstructed data with inverse FFT
		coeffs, err := fs.FFT(data, true)
		if err != nil {
			t.Fatal(err)
		}
		debugFrs("coeffs data", coeffs)
		// second half of all coefficients should be zero
		for i := fs.MaxWidth / 2; i < fs.MaxWidth; i++ {
			if !kzg_bls.EqualZero(&coeffs[i]) {
				t.Errorf("expected zero coefficient on index %d", i)
			}
		}
	}
	for scale := uint8(4); scale < 10; scale++ {
		for i := int64(0); i < 4; i++ {
			t.Run(fmt.Sprintf("scale_%d_i_%d", scale, i), func(t *testing.T) {
				testScale(i, scale, t)
			})
		}
	}
}
