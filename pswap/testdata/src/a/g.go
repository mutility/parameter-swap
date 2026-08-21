//go:build go1.27
package a

type SGM struct {}
func (SGM) Copy[T any](target, source T) {}
func (SGM) Copy2[T, U any](target T, source U) {}

type GSM[T any] struct {}
func (GSM[T]) Copy2[U any](target T, source U) {}

func genmethodtests() {
	var source, other D
	var another int
	sgm := SGM{}
	sgm.Copy(source, other) // want `passes 'source' as 'target' in call to func \(SGM\).Copy\[T any\]\(target T, source T\) \(position 0 vs 1\)`
	sgm.Copy2(source, other) // want `passes 'source' as 'target' in call to func \(SGM\).Copy2\[T, U any\]\(target T, source U\) \(position 0 vs 1\)`
	sgm.Copy2(source, another) // mismatched types due to instantiation

	gsm := GSM[D]{}
	gsm.Copy2(source, other) // want `passes 'source' as 'target' in call to func \(GSM\[T\]\).Copy2\[U any\]\(target T, source U\) \(position 0 vs 1\)`
	gsm.Copy2(source, another) // mismatched types due to instantiation
}

var _ = genmethodtests
