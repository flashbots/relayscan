package database

type MockDB struct{}

func (s *MockDB) SaveBidForSlot(relay string, slot uint64, parentHash, proposerPubkey string, respStatus uint64, respBid any, respError string) error {
	return nil
}
