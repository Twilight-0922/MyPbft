package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
	"strings"
)

func debugFrPtrs(msg string, values []*kzg_bls.Fr) {
	var out strings.Builder
	out.WriteString("---")
	out.WriteString(msg)
	out.WriteString("---\n")
	for i := range values {
		out.WriteString(fmt.Sprintf("#%4d: %s\n", i, kzg_bls.FrStr(values[i])))
	}
	fmt.Println(out.String())
}

func debugFrs(msg string, values []kzg_bls.Fr) {
	fmt.Println("---------------------------")
	var out strings.Builder
	for i := range values {
		out.WriteString(fmt.Sprintf("%s %d: %s\n", msg, i, kzg_bls.FrStr(&values[i])))
	}
	fmt.Print(out.String())
}
