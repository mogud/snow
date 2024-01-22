package container

type Pair[T, U any] struct {
	First  T
	Second U
}

type Triple[T, U, V any] struct {
	First  T
	Second U
	Third  V
}

type Tuple4[T0, T1, T2, T3 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
}

type Tuple5[T0, T1, T2, T3, T4 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
}

type Tuple6[T0, T1, T2, T3, T4, T5 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
	V5 T5
}

type Tuple7[T0, T1, T2, T3, T4, T5, T6 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
	V5 T5
	V6 T6
}

type Tuple8[T0, T1, T2, T3, T4, T5, T6, T7 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
	V5 T5
	V6 T6
	V7 T7
}

type Tuple9[T0, T1, T2, T3, T4, T5, T6, T7, T8 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
	V5 T5
	V6 T6
	V7 T7
	V8 T8
}

type Tuple10[T0, T1, T2, T3, T4, T5, T6, T7, T8, T9 any] struct {
	V0 T0
	V1 T1
	V2 T2
	V3 T3
	V4 T4
	V5 T5
	V6 T6
	V7 T7
	V8 T8
	V9 T9
}
