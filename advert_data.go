package dv

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/named-data/ndnd/dv/tlv"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

func writeToAdvertLog(msg string) {
	file, err := os.OpenFile("advert_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("로그 파일 열기 실패:", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(msg + "\n")
	if err != nil {
		fmt.Println("로그 파일 쓰기 실패:", err)
	}
}

func (a *advertModule) generate() {
	a.dv.mutex.Lock()
	defer a.dv.mutex.Unlock()

	// Increment sequence number
	a.seq++

	// Produce the advertisement
	name := a.dv.config.AdvertisementDataPrefix().
		Append(enc.NewTimestampComponent(a.bootTime)).
		WithVersion(a.seq)
	name, err := a.dv.client.Produce(ndn.ProduceArgs{
		Name:            name,
		Content:         a.dv.rib.Advert().Encode(),
		FreshnessPeriod: 10 * time.Second,
	})
	if err != nil {
		log.Error(a, "Failed to produce advertisement", "err", err)
	}

	advertisement := a.dv.rib.Advert()
	// fmt.Print("Advertisement Raw Data: ", advertisement)
	// content := advertisement.Encode()
	// content_size := len(content.Join())
	// fmt.Printf("[Advert Size] Encoded advertisement size: %d bytes\n", content_size)
	// // fmt.Printf("Advertisement Entries: Dest(%d), Nexthop(%d), Cost(%d), Othercost(%d)\n")
	// for _, entry := range advertisement.Entries {
	// 	fmt.Printf("        Destination: %s, NextHop: %s, Cost: %d\n",
	// 		entry.Destination.Name.String(),
	// 		entry.NextHop.Name.String(),
	// 		entry.Cost)
	// }

	// 인코딩
	ibf := advertisement.IBFprogram()
	payload := ibf.Encode().Join()
	size := len(payload)
	fmt.Printf("[Initial UDP Advert][Seq=%d] PayloadSize=%d bytes, Neighbors=%d\n",
		a.seq, size, len(a.dv.config.Neighbors))

	// 모든 이웃에게 UDP 전송
	for _, nbr := range a.dv.config.Neighbors {
		// 소켓 선택
		conn, ok := a.dv.conns[nbr.From]
		if !ok {
			log.Warn(a, "no UDP socket for", "from", nbr.From)
			continue
		}
		// 대상 주소 파싱
		u, err := url.Parse(nbr.Uri)
		if err != nil {
			log.Warn(a, "invalid neighbor URI", "uri", nbr.Uri, "err", err)
			continue
		}
		udpAddr, err := net.ResolveUDPAddr("udp4", u.Host)
		if err != nil {
			log.Warn(a, "invalid neighbor address", "host", u.Host, "err", err)
			continue
		}
		// 전송
		if _, err := conn.WriteToUDP(payload, udpAddr); err != nil {
			log.Warn(a, "initial UDP push failed", "to", nbr.Name, "err", err)
		} else {
			fmt.Printf("Initial advert pushed to %s via %s\n", nbr.Name, nbr.From)
		}
	}

	a.objDir.Push(name)
	a.objDir.Evict(a.dv.client)

	// Notify neighbors with sync for new advertisement
	go a.sendSyncInterest()
}

func (a *advertModule) generateToNei() { // push 방식으로 nei 마다
	fmt.Printf("generate to nei\n")
	a.dv.mutex.Lock()
	defer a.dv.mutex.Unlock()

	// Increment sequence number
	a.seq++

	for _, nbr := range a.dv.config.Neighbors {
		fmt.Printf("generate to nei // ns : %s \n", nbr.Name)

		advertisement := a.dv.rib.AdvertToNei(nbr.Name)

		//IBF program
		ibf := advertisement.IBFprogram()
		ibf_content := ibf.Encode()

		content_size := len(ibf_content.Join())
		// fmt.Printf("		[Advert Size]  %d bytes\n", content_size)
		// for i, cell := range ibf.Cells {
		// 	fmt.Printf("			[%d] Cell: %d → key: %d bytes, sig: %d, count: %d\n",
		// 		i, len(cell.Encode().Join()),
		// 		len(cell.KeyField), 1+enc.Nat(cell.SigField).EncodingLength(), 1+enc.Nat(cell.Count).EncodingLength())
		// }
		logMsg := fmt.Sprintf("[Advert Size][%s] %d bytes", nbr.Name, content_size)
		fmt.Println(logMsg)      // 콘솔에도 출력
		writeToAdvertLog(logMsg) // 파일에도 저장

		// 보낼 소켓 선택
		conn, ok := a.dv.conns[nbr.From]
		if !ok {
			log.Warn(a, "no socket for From", "from", nbr.From)
			continue
		}

		// 송신 대상 주소 파싱
		u2, err := url.Parse(nbr.Uri)
		if err != nil {
			log.Warn(a, "invalid neighbor URI, skip", "uri", nbr.Uri, "err", err)
			continue
		}
		udpAddr, err := net.ResolveUDPAddr("udp4", u2.Host)
		if err != nil {
			log.Warn(a, "invalid neighbor addr, skip", "host", u2.Host, "err", err)
			continue
		}

		// UDP 전송
		if _, err := conn.WriteToUDP(ibf_content.Join(), udpAddr); err != nil {
			log.Warn(a, "UDP push failed", "to", nbr.Name, "addr", udpAddr, "err", err)
		} else {
			fmt.Printf("Pushed advert to %s via %s\n", nbr.Name, nbr.From)
		}
	}
}

func (a *advertModule) dataFetch(nName enc.Name, bootTime uint64, seqNo uint64) {
	a.dv.mutex.Lock()
	defer a.dv.mutex.Unlock()

	if ns := a.dv.neighbors.Get(nName); ns == nil || ns.AdvertBoot != bootTime || ns.AdvertSeq != seqNo {
		return
	}

	// Fetch the advertisement
	advName := enc.LOCALHOP.
		Append(nName...).
		Append(enc.NewKeywordComponent("DV")).
		Append(enc.NewKeywordComponent("ADV")).
		Append(enc.NewTimestampComponent(bootTime)).
		WithVersion(seqNo)

	a.dv.client.Consume(advName, func(state ndn.ConsumeState) {
		if err := state.Error(); err != nil {
			log.Warn(a, "Failed to fetch advertisement", "name", state.Name(), "err", err)
			time.AfterFunc(1*time.Second, func() {
				a.dataFetch(nName, bootTime, seqNo)
			})
			return
		}

		//		log.Info(a, "|||||Fetched Adv Success||||| Fetched Adv:", "name", state.Name(), "content", state.Content())

		// Process the advertisement
		go a.dataHandler(nName, seqNo, state.Content())
	})
}

// Received advertisement Data
func (a *advertModule) dataHandler(nName enc.Name, seqNo uint64, data enc.Wire) {
	a.dv.mutex.Lock()
	defer a.dv.mutex.Unlock()

	// Check if this is the latest advertisement
	ns := a.dv.neighbors.Get(nName)
	if ns == nil {
		log.Warn(a, "Unknown advertisement", "name", nName)
		return
	}
	if ns.AdvertSeq != seqNo {
		log.Debug(a, "Old advertisement", "name", nName, "want", ns.AdvertSeq, "have", seqNo)
		return
	}
	// Parse the advertisement
	advert, err := tlv.ParseAdvertisement(enc.NewWireView(data), false)
	if err != nil {
		log.Error(a, "Failed to parse advertisement", "err", err)
		return
	}

	// Update the local advertisement list
	ns.Advert = advert
	go a.dv.updateRib(ns)
}

// 수신한 IBF 처리
func (a *advertModule) handlePushedAdvert(senderIBF *tlv.IBF, from *net.UDPAddr) {
	// fmt.Printf("handlePushedAdvert!!! 광고 수신 가능!!!\n")

	var senderName string
	senderName = a.dv.neighborByAddr[from.String()]
	fmt.Printf("Received UDP advertisement from %s (%s)\n", senderName, from.String())

	// 로컬 IBF 생성 (nexthop이 sender인 경우로만)
	localAdv := a.dv.rib.AdvertLocal(senderName)
	localIBF := localAdv.IBFprogram()

	// dIBF 생성
	dIBF := tlv.DIBFgenerate(senderIBF, localIBF)

	//dIBF decoding
	addEntry, withdrawEntry, err := dIBF.DIBFdecode()
	if err != nil {
		log.Error(a, "Failed to decode dIBF", "err", err)
	}

	// Build Advertisement from decoded entries
	addAdvert := &tlv.Advertisement{
		Entries: addEntry,
	}
	withdrawAdvert := &tlv.Advertisement{
		Entries: withdrawEntry,
	}

	log.Info(a, "Decoded add Advertisement", "nEntries", len(addAdvert.Entries))
	for _, entry := range addAdvert.Entries {
		fmt.Printf("         destination: %s , nexthop: %s , cost: %d\n",
			entry.Destination.Name.String(),
			entry.NextHop.Name.String(),
			entry.Cost)
	}

	log.Info(a, "Decoded withdraw Advertisement", "nEntries", len(withdrawAdvert.Entries))
	for _, entry := range withdrawAdvert.Entries {
		fmt.Printf("         destination: %s , nexthop: %s , cost: %d\n",
			entry.Destination.Name.String(),
			entry.NextHop.Name.String(),
			entry.Cost)
	}

	// Trigger our own advertisement if needed
	var dirty bool = false

	// add 수행
	for _, entry := range addAdvert.Entries {

		// fmt.Printf("         dest: %s , nexthop: %s , cost: %d \n",
		// 	entry.Destination.Name.String(),
		// 	entry.NextHop.Name.String(),
		// 	entry.Cost)

		// Update the local advertisement list
		prefix := entry.Destination.Name
		nexthop := entry.NextHop.Name // IBF 적용 이후에는 entry.NextHop.Name
		cost := entry.Cost

		a.dv.rib.Set(prefix, nexthop, cost)

		dirty = true
	}

	// Drop dead entries
	dirty = a.dv.rib.Prune() || dirty

	// If advert changed, increment sequence number
	if dirty {
		go a.dv.postUpdateRib()
	}

}
