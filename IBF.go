package tlv

import (
	"errors"
	"fmt"
	"reflect"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/twmb/murmur3"
)

const IBF_CellCount = 50

type IBF struct {
	Cells []*IBFEntry `tlv:"0xE1"`
}

// IBFEntry represents a cell in the IBF
type IBFEntry struct {
	KeyField []byte `tlv:"0xE2"`
	SigField uint64 `tlv:"0xE3"`
	Count    int64  `tlv:"0xE4"`
}

func NewIBF() *IBF {
	cells := make([]*IBFEntry, IBF_CellCount)
	for i := 0; i < IBF_CellCount; i++ {
		cells[i] = &IBFEntry{
			KeyField: []byte{},
			SigField: 0,
			Count:    0,
		}
	}
	return &IBF{Cells: cells}
}

func extractIBFIndexesFromKey(key []byte) [3]int {
	h1 := murmur3.SeedSum32(0xA1A1A1A1, key)
	h2 := murmur3.SeedSum32(0xB2B2B2B2, key)
	h3 := murmur3.SeedSum32(0xC3C3C3C3, key)

	return [3]int{
		int(h1 % uint32(IBF_CellCount)),
		int(h2 % uint32(IBF_CellCount)),
		int(h3 % uint32(IBF_CellCount)),
	}
}

/************************** IBF Program ****************************/
func (adv *Advertisement) IBFprogram() *IBF {
	fmt.Printf("IBF program / ")

	ibf := NewIBF()

	for _, entry := range adv.Entries {
		data := entry.Encode().Join()
		// hash32 := murmur3.Sum32(data)
		// hash32 := crc32.ChecksumIEEE(data)
		// indexes := extractIBFIndexes(hash32)

		key := data
		// sig := uint64((hash32 >> 24) & 0xFFFF)
		// fmt.Printf("	[%d]번째 : [%d][%d][%d] key: %s\n", i+1, h1, h2, h3, entry.Destination.Name)
		indexes := extractIBFIndexesFromKey(data)
		sigHash := murmur3.SeedSum32(0xD4D4D4D4, data)
		sig := uint64((sigHash >> 24) & 0xFFFF)

		h1, h2, h3 := indexes[0], indexes[1], indexes[2]

		ibf.InsertAt(h1, key, sig)
		ibf.InsertAt(h2, key, sig)
		ibf.InsertAt(h3, key, sig)

	}
	return ibf
}

func (ibf *IBF) InsertAt(idx int, key []byte, sig uint64) {
	cell := ibf.Cells[idx]

	if cell.KeyField == nil {
		cell.KeyField = make([]byte, len(key))
	}

	cell.KeyField = xorBytesNew(cell.KeyField, key)
	cell.SigField = cell.SigField ^ sig
	cell.Count++
}

func xorBytesNew(a, b []byte) []byte {
	n := max(len(a), len(b)) // 둘 중 긴 길이에 맞춘다
	result := make([]byte, n)

	// 뒤에서부터 맞춰서 복사
	copy(result[n-len(a):], a)
	for i := 0; i < len(b); i++ {
		result[n-len(b)+i] ^= b[i]
	}
	return result
}

/************************** dIBF generate ****************************/
func DIBFgenerate(senderIBF, localIBF *IBF) *IBF {
	fmt.Printf(" dIBF 생성 ")

	// PrintIBF("Sender IBF", senderIBF)
	// PrintIBF("Local IBF", localIBF)

	dibf := NewIBF()

	if len(senderIBF.Cells) == 0 || len(localIBF.Cells) == 0 {
		log.Error(nil, "DIBFgenerate: 셀 길이가 0",
			"remoteLen", fmt.Sprintf("%d", len(senderIBF.Cells)),
			"localLen", fmt.Sprintf("%d", len(localIBF.Cells)))
		return &IBF{Cells: []*IBFEntry{}}
	}

	for i := 0; i < IBF_CellCount; i++ {
		cell1 := senderIBF.Cells[i]
		cell2 := localIBF.Cells[i]

		// key: XOR (길이 다를 수 있음)
		dKey := xorBytesNew(cell1.KeyField, cell2.KeyField)
		dSig := cell1.SigField ^ cell2.SigField
		dCount := cell1.Count - cell2.Count

		// 새 셀에 설정
		dibf.Cells[i] = &IBFEntry{
			KeyField: dKey,
			SigField: dSig,
			Count:    dCount,
		}
	}

	// PrintIBF("dIBF (차이)", dibf)
	// return &IBF{Cells: dcells}
	return dibf
}

func PrintIBF(name string, ibf *IBF) {
	fmt.Printf("\n========== %s ==========\n", name)
	for i, cell := range ibf.Cells {
		fmt.Printf("[%02d] Count: %d, Sig: 0x%x, Key: %x\n", i, cell.Count, cell.SigField, cell.KeyField)
	}
	fmt.Println("===============================\n")
}

/************************** dIBF Decoding ****************************/
func (ibf *IBF) DIBFdecode() ([]*AdvEntry, []*AdvEntry, error) {
	fmt.Println("Decoding start!")

	var (
		sigMismatchCount int
		t1CaseCount      int
		t2CaseCount      int
	)

	pureList := updatePureList(ibf)

	var decodedAddEntries []*AdvEntry
	var decodedWithdrawEntries []*AdvEntry

	for {
		found := false

		for i := 0; i < len(ibf.Cells); i++ {

			if !pureList[i] { // pureList 0이면 스킵
				continue
			}

			cell := ibf.Cells[i]
			key := trimPadding(cell.KeyField) //adventry.Encode().Join()

			if len(key) == 0 { // KeyField가 비어있으면 스킵
				pureList[i] = false
				continue
			}

			adventry, err := ParseAdvEntry(enc.NewWireView(enc.Wire{key}), false)
			if err != nil {
				return nil, nil, fmt.Errorf(" fail Parse : %w \n ", err)
			}

			// hash32 := murmur3.Sum32(key)
			// hash32 := crc32.ChecksumIEEE(key)
			// indexes := extractIBFIndexes(hash32)
			indexes := extractIBFIndexesFromKey(key)

			h1, h2, h3 := indexes[0], indexes[1], indexes[2]

			// fmt.Printf(" 		 h1, h2, h3 : [%d],[%d],[%d] \n", h1, h2, h3)

			// 해당 key의 sig
			sigHash := murmur3.SeedSum32(0xD4D4D4D4, key)
			expectedSig := uint64((sigHash >> 24) & 0xFFFF)
			// expectedSig := uint64(uint16((hash32 >> 24) & 0xFFFF))

			// Fake Pure Cell 체크
			isFakePure := false

			if cell.SigField != expectedSig {
				sigMismatchCount++
				isFakePure = true
				continue
			}

			// 2. T1 case: 해당 셀이 자기 자신의 해시 index에 속하지 않음
			if i != h1 && i != h2 && i != h3 {
				t1CaseCount++
				isFakePure = true
			}

			if isFakePure {
				pureList[i] = false
				continue
			}

			// T2 case 확인 후 append
			if cell.Count == 1 {
				// withdraw 리스트에 존재한다면 T2Case
				if contains(decodedWithdrawEntries, adventry) {
					t2CaseCount++
					decodedWithdrawEntries = removeEntry(decodedWithdrawEntries, adventry)
				} else if !contains(decodedAddEntries, adventry) {
					decodedAddEntries = append(decodedAddEntries, adventry)
				}
			} else if cell.Count == -1 {
				// add 리스트에 존재한다면 T2Case
				if contains(decodedAddEntries, adventry) {
					t2CaseCount++
					decodedAddEntries = removeEntry(decodedAddEntries, adventry)
				} else if !contains(decodedWithdrawEntries, adventry) {
					decodedWithdrawEntries = append(decodedWithdrawEntries, adventry)
				}
			}
			found = true

			// XOR로 주변 셀에서 제거
			removeKeyFromCell(ibf.Cells[h1], key, expectedSig)
			removeKeyFromCell(ibf.Cells[h2], key, expectedSig)
			removeKeyFromCell(ibf.Cells[h3], key, expectedSig)

			// pureList 갱신
			pureList = updatePureList(ibf)
		}

		if !found {
			break
		}
	}

	// 디코딩 끝났는데 남은 셀 존재 → 실패
	leftover := false
	for i, cell := range ibf.Cells {
		if cell.Count != 0 {
			leftover = true
			fmt.Printf("[Leftover Cell %02d] Count: %d, Sig: 0x%x, Key: %x\n", i, cell.Count, cell.SigField, cell.KeyField)
		}
	}
	if leftover {
		return decodedAddEntries, decodedWithdrawEntries, errors.New("cannot fully decode IBF, leftover entries exist")
	}

	return decodedAddEntries, decodedWithdrawEntries, nil
}

func removeKeyFromCell(cell *IBFEntry, key []byte, sig uint64) {
	n := max(len(cell.KeyField), len(key))

	// 패딩해서 동일 길이로 맞추기
	paddedCellKey := make([]byte, n)
	paddedKey := make([]byte, n)

	copy(paddedCellKey[n-len(cell.KeyField):], cell.KeyField)
	copy(paddedKey[n-len(key):], key)

	// XOR
	for i := 0; i < n; i++ {
		paddedCellKey[i] ^= paddedKey[i]
	}

	// KeyField 업데이트
	cell.KeyField = paddedCellKey

	// SigField는 그냥 XOR
	cell.SigField ^= sig

	// Count 업데이트
	if cell.Count > 0 {
		cell.Count--
	} else {
		cell.Count++
	}
}

func trimPadding(key []byte) []byte {
	idx := 0
	for idx < len(key) && key[idx] == 0 {
		idx++
	}
	return key[idx:]
}

/// T2 error 관련

func entryEquals(a, b *AdvEntry) bool {
	return reflect.DeepEqual(a.Destination.Name, b.Destination.Name) &&
		reflect.DeepEqual(a.NextHop.Name, b.NextHop.Name) &&
		a.Cost == b.Cost
}

func contains(entries []*AdvEntry, target *AdvEntry) bool {
	for _, e := range entries {
		if entryEquals(e, target) {
			return true
		}
	}
	return false
}

func updatePureList(ibf *IBF) []bool {
	pureList := make([]bool, len(ibf.Cells))

	for i, cell := range ibf.Cells {
		if cell.Count != 1 && cell.Count != -1 {
			continue
		}

		key := trimPadding(cell.KeyField)
		if len(key) == 0 {
			continue
		}

		// hash32 := murmur3.Sum32(key)
		// hash32 := crc32.ChecksumIEEE(key)
		// expectedSig := uint64(uint16((hash32 >> 24) & 0xFFFF))
		sigHash := murmur3.SeedSum32(0xD4D4D4D4, key)
		expectedSig := uint64((sigHash >> 24) & 0xFFFF)

		if cell.SigField != expectedSig {
			continue
		}

		// indexes := extractIBFIndexes(hash32)
		indexes := extractIBFIndexesFromKey(key)
		h1, h2, h3 := indexes[0], indexes[1], indexes[2]

		if i == h1 || i == h2 || i == h3 {
			pureList[i] = true
		}
	}

	return pureList
}

func removeEntry(list []*AdvEntry, target *AdvEntry) []*AdvEntry {
	for i, entry := range list {
		if entryEquals(entry, target) {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func extractIBFIndexes(hash32 uint32) [3]int {
	const mask5bit = 0x1F

	h1 := int((hash32>>0)&mask5bit) % IBF_CellCount
	h2 := int((hash32>>5)&mask5bit) % IBF_CellCount
	h3 := int((hash32>>12)&mask5bit) % IBF_CellCount

	return [3]int{h1, h2, h3}
}
