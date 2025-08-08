package main

import (
	"flag"
	"fmt"
	"net"

	utils "github.com/epyon0/goUtils"
)

var debug *bool
var port *uint

type dnsPacket struct {
	data dnsData
	rr   resourceRecord
	q    question
}

type dnsData struct {
	ID      uint16
	FLAGS   uint16
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
}

type resourceRecord struct {
	NAME     []byte
	TYPE     uint16
	CLASS    uint16
	TTL      uint32
	RDLENGTH uint16
	RDATA    []byte
}

type question struct {
	QNAME  []byte
	QTYPE  uint16
	QCLASS uint16
}

func main() {
	debug = flag.Bool("v", false, "Enable verbose output")
	port = flag.Uint("p", 53, "Define UDP port to use")
	flag.Parse()
	*port = uint(uint16(*port))

	utils.Debug(fmt.Sprintf("Creating socket [0.0.0.0:%d]", *port), *debug)
	sock, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", *port))
	utils.Er(err)

	utils.Debug("Creating connection", *debug)
	conn, err := net.ListenUDP("udp4", sock)
	utils.Er(err)
	defer conn.Close()

	buf := make([]byte, 4096)

	for {
		utils.Debug("Reading from connection", *debug)
		n, addr, err := conn.ReadFromUDP(buf)
		utils.Er(err)
		data := buf[:n]

		utils.Debug(fmt.Sprintf("Read %d bytes from %d.%d.%d.%d:%d", n, addr.IP[0], addr.IP[1], addr.IP[2], addr.IP[3], addr.Port), *debug)
		utils.Debug(fmt.Sprintf("String Data: \n{\n%s\n}", string(data)), *debug)
		utils.Debug(fmt.Sprintf("Binary Data: \n{\n%s\n}", utils.WalkByteSlice(data)), *debug)

		// parse data
		if data[2]&0x78 == 0 || data[2]&0x78 == 1 { // QUERY or IQUERY
			var packet dnsPacket
			packet.data.ID = (uint16(data[0]) << 8) + uint16(data[1])
			packet.data.FLAGS = (uint16(data[2]) << 8) + uint16(data[3])
			packet.data.QDCOUNT = (uint16(data[4]) << 8) + uint16(data[5])
			packet.data.ANCOUNT = (uint16(data[6]) << 8) + uint16(data[7])
			packet.data.NSCOUNT = (uint16(data[8]) << 8) + uint16(data[9])
			packet.data.ARCOUNT = (uint16(data[10]) << 8) + uint16(data[11])

			for i := 12; i < len(data); i++ {
				if data[i] == 0 {
					packet.q.QNAME = data[12 : i+1]
				}
				length := int(data[i])
				i += length
			}

			// send response
			utils.Debug("Sending response", *debug)
			_, err = conn.WriteToUDP(data, addr)
			utils.Er(err)
		}
	}
}
