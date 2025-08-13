package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	utils "github.com/epyon0/goUtils"
	toml "github.com/pelletier/go-toml"
)

var debug *bool
var port *uint
var config *string

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

type tomlConfig struct {
	ttl   uint16
	a     [][2]string
	ns    [][2]string
	cname [][2]string
	ptr   [][2]string
	mx    [][2]string
	txt   [][2]string
}

func main() {
	filePath, err := os.Executable()
	utils.Er(err)

	debug = flag.Bool("v", false, "Enable verbose output")
	port = flag.Uint("p", 53, "Define alternate UDP port")
	config = flag.String("c", fmt.Sprintf("%s/config.toml", filepath.Dir(filePath)), "Define alternate configuration file")
	flag.Parse()
	*port = uint(uint16(*port))

	file, err := os.Open(*config)
	utils.Er(err)
	defer file.Close()

	configBytes, err := io.ReadAll(file)
	utils.Er(err)

	toml.LoadBytes(configBytes)

	utils.Debug(fmt.Sprintf("Creating socket [0.0.0.0:%d]", *port), *debug)
	sock, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", *port))
	utils.Er(err)

	utils.Debug("Creating connection", *debug)
	conn, err := net.ListenUDP("udp4", sock)
	utils.Er(err)
	conn.Close()

	buf := make([]byte, 4096)

	for {
		utils.Debug("Reading from connection", *debug)
		n, addr, err := conn.ReadFromUDP(buf)
		utils.Er(err)
		data := buf[:n]

		utils.Debug(fmt.Sprintf("Read %d bytes from %d.%d.%d.%d:%d", n, addr.IP[0], addr.IP[1], addr.IP[2], addr.IP[3], addr.Port), *debug)
		utils.Debug(fmt.Sprintf("String Data:\n{\n%s\n}", string(data)), *debug)
		utils.Debug(fmt.Sprintf("Hex Data:\n{\n%s\n}", utils.WalkByteSlice(data)), *debug)
		utils.Debug(fmt.Sprintf("Binary Data:\n{\n%s\n}", utils.DumpByteSlice(data)), *debug)

		// parse data
		if data[2]&0x78 == 0 || data[2]&0x78 == 1 { // QUERY or IQUERY
			if data[2]&0x78 == 0 {
				utils.Debug("QUERY", *debug)
			}
			if data[2]&0x78 == 1 {
				utils.Debug("IQUERY", *debug)
			}

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

			// Populate RR here
			switch packet.q.QTYPE {
			case 1: // A
			// look though config for A record that matches packet.q.QNAME

			case 2: // NS

			case 5: // CNAME

			case 12: // PTR

			case 15: // MX

			case 16: // TXT

			}

			// send response

			var reply []byte

			reply = append(reply, byte(packet.data.ID>>8), byte(packet.data.ID))
			flagByte := packet.data.FLAGS & 0x7800
			flagByte += 0x8000 //QR - Response
			flagByte += 0x400  // AA
			reply = append(reply, byte(flagByte>>8), byte(flagByte))
			reply = append(reply, byte(packet.data.QDCOUNT>>8), byte(packet.data.QDCOUNT))
			reply = append(reply, byte(packet.data.ANCOUNT>>8), byte(packet.data.ANCOUNT))
			reply = append(reply, byte(packet.data.NSCOUNT>>8), byte(packet.data.NSCOUNT))
			reply = append(reply, byte(packet.data.ARCOUNT>>8), byte(packet.data.ARCOUNT))

			// fill in RR first
			for i := 0; i < len(packet.rr.NAME); i++ {
				reply = append(reply, packet.rr.NAME[i])
			}
			reply = append(reply, byte(packet.rr.TYPE>>8), byte(packet.rr.TYPE))
			reply = append(reply, byte(packet.rr.CLASS>>8), byte(packet.rr.CLASS))
			reply = append(reply, byte(packet.rr.TTL>>24), byte(packet.rr.TTL>>16), byte(packet.rr.TTL>>8), byte(packet.rr.TTL))
			reply = append(reply, byte(packet.rr.RDLENGTH>>8), byte(packet.rr.RDLENGTH))
			for i := 0; i < len(packet.rr.RDATA); i++ {
				reply = append(reply, packet.rr.RDATA[i])
			}

			for i := 0; i < len(packet.q.QNAME); i++ {
				reply = append(reply, packet.q.QNAME[i])
			}
			reply = append(reply, byte(packet.q.QTYPE>>8), byte(packet.q.QTYPE))
			reply = append(reply, byte(packet.q.QCLASS>>8), byte(packet.q.QCLASS))

			utils.Debug("Sending response", *debug)
			//_, err = conn.WriteToUDP(data, addr)
			//utils.Er(err)
		}
	}
}
