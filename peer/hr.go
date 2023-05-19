package peer

type HR interface {
	CreateHR(id uint, secret string, num uint)
	// GetHR() *HR
}
