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

package layers

/*
0000   54 53 50 2a 1d 00 92 00 01 00 00 00 18 00 80 df
0010   00 00 38 00 b0 3d d9 0c 1d 00 00 00 00 00 00 00
0020   00 00 00 00 01 00 00 00 00 00 00 00 10 02 81 df
0030   00 00 39 00 b0 3d d9 0c 1d 00 00 00 00 00 00 00
0040   00 00 00 00 d8 e4 e8 e4 f0 e4 ec e4 e8 e4 ec e4
0050   e4 e4 f8 e4 e8 e4 f0 e4 dc e4 d0 e4 d4 e4 d4 e4
0060   d8 e4 dc e4 f4 e4 dc e4 e4 e4 e0 e4 f4 e4 dc e4
0070   e4 e4 f8 e4 fc e4 e0 e4 04 e5 f8 e4 c0 e4 dc e4
0080   d0 e4 b0 e4 f4 e4 ec e4 e4 e4 d0 e4 f4 e4 dc e4
0090   ec e4 e4 e4 d4 e4 dc e4 e8 e4 d8 e4 f8 e4 e0 e4
00a0   e4 e4 f4 e4 bc e4 f0 e4 cc e4 e4 e4 d4 e4 d4 e4
00b0   ec e4 f8 e4 ec e4 f8 e4 d0 e4 cc e4 f4 e4 c8 e4
00c0   38 e5 24 e5 fc e4 28 e5 b8 e4 d0 e4 d0 e4 cc e4
00d0   18 e5 0c e5 fc e4 20 e5 98 e4 c0 e4 dc e4 bc e4
00e0   30 e5 1c e5 fc e4 38 e5 ac e4 cc e4 c8 e4 b0 e4
00f0   2c e5 14 e5 f4 e4 18 e5 b8 e4 dc e4 dc ec a0 e4
0100   90 25 10 1b 50 23 10 1e 44 25 44 26 88 25 a8 25
0110   e4 24 d0 24 20 26 60 25 18 27 e0 26 c0 26 04 27
0120   4c 26 50 26 c4 26 80 26 e8 26 18 27 40 26 a4 26
0130   34 26 10 26 a0 26 80 26 a0 26 d0 26 20 26 58 26
0140   60 26 14 26 08 27 c0 26 ac 26 f8 26 08 26 5c 26
0150   40 26 0c 26 c0 26 a0 26 b4 26 d8 26 40 26 50 26
0160   40 26 40 26 a4 26 8c 26 80 26 a4 26 24 26 40 26
0170   4c 26 30 26 80 26 80 26 64 26 80 26 9c e7 08 13
0180   a8 ec c4 e8 3c e5 84 e6 6c e5 e8 e5 78 e6 d4 e5
0190   cc e5 34 e6 34 e4 ec e4 fc e3 ec e3 cc e4 74 e4
01a0   ac e4 e4 e4 0c e4 44 e4 4c e4 24 e4 f8 e4 bc e4
01b0   bc e4 fc e4 3c e4 74 e4 8c e4 70 e4 d0 e4 b8 e4
01c0   94 e4 d4 e4 54 e4 70 e4 98 e4 88 e4 ec e4 c0 e4
01d0   a8 e4 cc e4 8c e4 a0 e4 ac e4 94 e4 cc e4 cc e4
01e0   9c e4 b8 e4 b8 e4 a4 e4 cc e4 c4 e4 e0 e4 e8 e4
01f0   e8 e4 e4 e4 bc e4 cc e4 b8 e4 d4 e4 cc e4 d4 e4
0200   c8 e4 b4 e4 dc e4 d4 e4 f0 e4 ec e4 d8 e4 e8 e4
0210   cc e4 d4 e4 f4 e4 cc e4 bc e4 d0 e4 cc e4 ec e4
0220   e0 e4 f0 e4 dc e4 d4 e4 bc e4 cc e4 b8 e4 c8 e4
0230   c4 e4 bc e4 bc e4 c8 e4 a0 e4 b8 e4 9c e4 b0 e4
0240   ac e4 ac e4 49 62 20 12

========
MLink
54 53 50 2a 1d 00 92 00 01 00 00 00
========
MStream fragment
len = 24 [0:2]
18 00
flags + subtype [2]
80
device id [3]
df
fragment offset [4:6]
00 00
fragment id [6:8]
38 00
====
MStream trigger
device serial [0:4]
b0 3d d9 0c
event num [4:7]
1d 00 00
unused [7]
00
timestamp sec [8:12]
00 00 00 00
timestamp nsec + flags [12:16]
00 00 00 00
low ch [16:20]
01 00 00 00
hi ch [20:24]
00 00 00 00
========
MStream fragment
len = 528 [0:2]
10 02
flags + subtype [2]
81
device id [3]
df
fragment offset [4:6]
00 00
fragment id [6:8]
39 00
====
MStream data
device serial [0:4]
b0 3d d9 0c
event num [4:7]
1d 00 00
channel num [7]
00
data [8:]
00 00 00 00 00 00 00 00 d8 e4
0070   e8 e4 f0 e4 ec e4 e8 e4 ec e4 e4 e4 f8 e4 e8 e4
0080   f0 e4 dc e4 d0 e4 d4 e4 d4 e4 d8 e4 dc e4 f4 e4
0090   dc e4 e4 e4 e0 e4 f4 e4 dc e4 e4 e4 f8 e4 fc e4
00a0   e0 e4 04 e5 f8 e4 c0 e4 dc e4 d0 e4 b0 e4 f4 e4
00b0   ec e4 e4 e4 d0 e4 f4 e4 dc e4 ec e4 e4 e4 d4 e4
00c0   dc e4 e8 e4 d8 e4 f8 e4 e0 e4 e4 e4 f4 e4 bc e4
00d0   f0 e4 cc e4 e4 e4 d4 e4 d4 e4 ec e4 f8 e4 ec e4
00e0   f8 e4 d0 e4 cc e4 f4 e4 c8 e4 38 e5 24 e5 fc e4
00f0   28 e5 b8 e4 d0 e4 d0 e4 cc e4 18 e5 0c e5 fc e4
0100   20 e5 98 e4 c0 e4 dc e4 bc e4 30 e5 1c e5 fc e4
0110   38 e5 ac e4 cc e4 c8 e4 b0 e4 2c e5 14 e5 f4 e4
0120   18 e5 b8 e4 dc e4 dc ec a0 e4 90 25 10 1b 50 23
0130   10 1e 44 25 44 26 88 25 a8 25 e4 24 d0 24 20 26
0140   60 25 18 27 e0 26 c0 26 04 27 4c 26 50 26 c4 26
0150   80 26 e8 26 18 27 40 26 a4 26 34 26 10 26 a0 26
0160   80 26 a0 26 d0 26 20 26 58 26 60 26 14 26 08 27
0170   c0 26 ac 26 f8 26 08 26 5c 26 40 26 0c 26 c0 26
0180   a0 26 b4 26 d8 26 40 26 50 26 40 26 40 26 a4 26
0190   8c 26 80 26 a4 26 24 26 40 26 4c 26 30 26 80 26
01a0   80 26 64 26 80 26 9c e7 08 13 a8 ec c4 e8 3c e5
01b0   84 e6 6c e5 e8 e5 78 e6 d4 e5 cc e5 34 e6 34 e4
01c0   ec e4 fc e3 ec e3 cc e4 74 e4 ac e4 e4 e4 0c e4
01d0   44 e4 4c e4 24 e4 f8 e4 bc e4 bc e4 fc e4 3c e4
01e0   74 e4 8c e4 70 e4 d0 e4 b8 e4 94 e4 d4 e4 54 e4
01f0   70 e4 98 e4 88 e4 ec e4 c0 e4 a8 e4 cc e4 8c e4
0200   a0 e4 ac e4 94 e4 cc e4 cc e4 9c e4 b8 e4 b8 e4
0210   a4 e4 cc e4 c4 e4 e0 e4 e8 e4 e8 e4 e4 e4 bc e4
0220   cc e4 b8 e4 d4 e4 cc e4 d4 e4 c8 e4 b4 e4 dc e4
0230   d4 e4 f0 e4 ec e4 d8 e4 e8 e4 cc e4 d4 e4 f4 e4
0240   cc e4 bc e4 d0 e4 cc e4 ec e4 e0 e4 f0 e4 dc e4
0250   d4 e4 bc e4 cc e4 b8 e4 c8 e4 c4 e4 bc e4 bc e4
0260   c8 e4 a0 e4 b8 e4 9c e4 b0 e4 ac e4 ac e4
========
MLink CRC
49 62 20 12
*/
