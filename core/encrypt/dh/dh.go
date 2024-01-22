package dh

var dh = &DH{0xFFFFFFFFFFFFFFA1, 3}

func PublicKeyOf(privateKey uint64) uint64 {
	return dh.PublicKeyOf(privateKey)
}

func LocalKey(privateKey, anotherPublicKey uint64) uint64 {
	return dh.LocalKey(privateKey, anotherPublicKey)
}

type DH struct {
	P uint64
	A uint64
}

func (ss *DH) PublicKeyOf(privateKey uint64) uint64 {
	return ss.powModP(ss.A, privateKey)
}

func (ss *DH) LocalKey(privateKey, anotherPublicKey uint64) uint64 {
	return ss.powModP(anotherPublicKey, privateKey)
}

func (ss *DH) mulModP(a, b uint64) uint64 {
	var m uint64 = 0
	for b > 0 {
		if b&1 > 0 {
			t := ss.P - a
			if m >= t {
				m -= t
			} else {
				m += a
			}
		}
		if a >= ss.P-a {
			a = a*2 - ss.P
		} else {
			a = a * 2
		}
		b >>= 1
	}
	return m
}

func (ss *DH) powModPImpl(a, b uint64) uint64 {
	if b == 1 {
		return a
	}
	t := ss.powModP(a, b>>1)
	t = ss.mulModP(t, t)
	if b%2 > 0 {
		t = ss.mulModP(t, a)
	}
	return t
}

func (ss *DH) powModP(a, b uint64) uint64 {
	if a > ss.P {
		a %= ss.P
	}
	return ss.powModPImpl(a, b)
}
