/*
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package reg

type RegPacket struct {
	Type uint16
	Seq uint16
	Src uint16
	Dst uint16
}

const (
	CtrlReg = 0x40
	CtrlDefault = 0x0000

	WinReg = 0x41
	StatusReg = 0x42 // read only
	WinDefault = 0xFFFF

	TrigMultReg = 0x43
	SerialIDHiReg = 0x46 // read only
	LiveMagic = 0x48

	ChDpmKsReg = 0x4A // read only

	TemperatureReg = 0x4B // read only
	FwVerReg = 0x4C // read only
    FwRevReg = 0x4D // read only

    SerialIDReg = 0x4E // read only
    MstreamCfg = 0x4F // read only

	TsReadA = 0x50
	TsSetReg64 = 0x54
	TsReadB = 0x58
	TsReadReg64 = 0x5C
)


type Device struct {
	hardwareAccessAllowed bool
	mstreamMultiAck uint16
	mstreamHwBufSize uint16
	fwVer uint16
	fwRev uint16
}

func (d *Device) regRead(regNumber int) uint16 {
	return 0
}

func (d *Device) regWrite(regNumber int, value uint16) {
}

func (d *Device) ctrlExchangeSingle(tx RegPacket) (rx *RegPacket, err error) {
	return nil, nil
}


/*
RegIoPacket MlinkDevice::ctrlExchangeSingle(const RegIoPacket &tx_in)
{
    int retry_count = 0;
    int seq = nextTxPacketSequenceNumber();
    while (enableState) {
        RegIoPacket tx(tx_in);
        tx.seq = seq;
        try {
            //            if (link.getCtrlPacketCount() > 0) {
            //                qWarning() << "MlinkDevice::ctrlExchangeSingle(): link.getCtrlPacketCount()) > 0";
            //                const PacketList &ctrlJunk = link.getCtrlPackets(link.getCtrlPacketCount(), 0);
            //                qDebug() << "ctrlJunk = " << ctrlJunk;
            //                throw std::runtime_error("protocol error");
            //            }
            link.rawSend(tx);
            double random_delay = MlinkPacketProtocol::random_delay(retry_count);
            const RegIoPacket &rx = link.getCtrlPacketBySeq(tx.seq, MlinkPacketProtocol::getTimeout() * random_delay);
            if (rx.seq == tx.seq) {
                validateIoAck(tx, rx);
                //                    qDebug() << "seq " << rx.seq << "Ok";
                return rx;
            } else {
                throw std::runtime_error("MlinkDevice::ctrlExchangeSingle(): Receive timeout");
            }
        }
        catch (std::runtime_error &e) {
            retry_count++;
            {
                //                qDebug() << QString("ctrlExchangeSingle failed: %1").arg(e.what());
                //                qDebug() << "TX was: " << tx;
                if (retry_count > MlinkPacketProtocol::getMaxRetryCount()) {
                    //                    qWarning() << QString("ctrlExchangeSingle failed after %1 attempts: %2").arg(retry_count).arg(e.what());
                    if (onlineState)
                        qWarning() << QString("%1: Going offline after %2 retry attempts").arg(getIdent()).arg(retry_count);
                    onlineState = false;
                    throw;
//                    break;
                }
            }
            //            qDebug() << QString("Retry #%1").arg(retry_count);
        }
    }
    return RegIoPacket();
}

void RegIOMLink::rawSend(const RegIoPacket &pkt)
{
//    int packetSize = 4 * (ML_FRAME_HEADER_WORDS + ML_FRAME_TRAILER_WORDS);
    std::vector<quint16> v;
    v.push_back(pkt.type);
    v.push_back(ML_FRAME_SYNC);
    v.push_back(pkt.seq);
    v.push_back((ML_FRAME_HEADER_WORDS + ML_FRAME_TRAILER_WORDS) + pkt.data.size());
    v.push_back(pkt.src);
    v.push_back(pkt.dst);
    QByteArray buf(reinterpret_cast<const char *>(&v[0]), 2 * v.size());
    const int dataSize = pkt.data.size();
    QByteArray bufData(reinterpret_cast<const char *>(&pkt.data[0]), sizeof(PacketRawDataType) * dataSize);
    buf.append(bufData);
#if defined(_MSC_VER) || defined (_WIN32)
    quint32 checksum = 0;
#else
    quint32 checksum = crc32(0, reinterpret_cast<const Bytef *>(buf.data()), buf.size());
#endif
    buf.append(reinterpret_cast<const char *>(&checksum), 4);
    if (deviceAddress.isNull()) {
        QString str("Device address not set");
        qWarning() << str;
        throw std::runtime_error(str.toStdString().c_str());
    }
    qint64 rc = socket->writeDatagram(buf, deviceAddress, ML_UDP_PORT);
//    qint64 rc = socket->write(buf);
//    {
//        //  printf("tx: size=%d\n", data.data.size());
//        std::ostringstream ost;
//        ost << "RegIOMLink::rawSend() " << data;
//        log_debug(0, ost.str());
//    }
    if (rc != buf.size()) {
        throw std::runtime_error(QString("Frame send failed: %1").arg(socket->errorString()).toStdString());
    }
//    socket->waitForBytesWritten(1000);
//    socket->waitForReadyRead(1000);
    socket->flush();
}

*/
