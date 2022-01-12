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

package mstream


/*
go-adc
0000   54 53 50 2a 00 00 06 00 fe fe 01 00 00 00 40 01
0010   00 00 10 00 00 00 00 00

AFI
0000   54 53 50 2a 0d 00 09 00 00 00 01 00 0c 00 40 01
0010   00 00 1b 00 00 00 1e 00 00 00 20 00 00 00 22 00
0020   00 00 00 00

===
MLink 12 bytes
Sync
54 53
Type MStream
50 2a
Seq
0d 00
Len
09 00
Dst
00 00
Src
01 00
===
MStream 8 bytes
Len
0c 00 // len of payload 12 bytes
Subtype + Flags
40
DeviceID
01
FragmentOffset
00 00
FragmentID
1b 00
===
MLink CRC 4 bytes
00 00 00 00
*/