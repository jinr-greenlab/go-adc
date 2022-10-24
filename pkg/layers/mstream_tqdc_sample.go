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
0000   54 53 50 2a 67 5c 2d 01 01 00 00 00 e8 00 80 d6
0010   00 00 ca 93 60 6b d9 0c 01 00 00 00 09 5a 50 63
0020   82 c6 79 80 18 00 00 00 c1 05 00 20 4a 2b 00 40
0030   03 00 00 30 c1 05 00 21 02 00 00 31 00 00 00 70
0040   b8 00 00 10 00 00 b4 00 40 fb 34 fb 34 fb 38 fb
0050   40 fb 48 fb 38 fb 34 fb 34 fb 40 fb 3c fb 50 fb
0060   40 fb 44 fb 2c fb 34 fb 44 fb 34 fb 40 fb 48 fb
0070   3c fb 48 fb 44 fb 3c fb 3c fb 4c fb 44 fb 40 fb
0080   40 fb 38 fb 3c fb 38 fb e4 fb ec fe f8 03 40 08
0090   c0 09 fc 08 d8 07 74 07 e4 07 7c 08 b8 08 90 08
00a0   28 08 68 06 24 02 38 fd 4c fa 24 fa 44 fb 14 fc
00b0   0c fc 7c fb 08 fb 04 fb 38 fb 60 fb 60 fb 4c fb
00c0   34 fb 2c fb 38 fb 38 fb 48 fb 3c fb 30 fb 3c fb
00d0   3c fb 50 fb 34 fb 40 fb 3c fb 34 fb 2c fb 34 fb
00e0   40 fb 40 fb 38 fb 40 fb 30 fb 2c fb 28 fb 30 fb
00f0   34 fb 34 fb 30 fb 24 fb 24 fb 34 fb e8 00 80 d6
0100   00 00 cb 93 60 6b d9 0c 02 00 00 00 09 5a 50 63
0110   82 78 f4 80 18 00 00 00 f1 1c 00 20 b7 2d 00 40
0120   03 10 00 30 f1 1c 00 21 02 10 00 31 00 00 00 70
0130   b8 00 00 10 00 00 b4 00 38 fb 48 fb 4c fb 50 fb
0140   44 fb 34 fb 38 fb 44 fb 48 fb 54 fb 58 fb 40 fb
0150   34 fb 44 fb 4c fb 50 fb 54 fb 50 fb 4c fb 4c fb
0160   50 fb 44 fb 50 fb 48 fb 34 fb 48 fb 4c fb 48 fb
0170   50 fb 4c fb 48 fb 40 fb 48 fb 58 fb 44 fc dc ff
0180   f4 04 c8 08 bc 09 bc 08 ac 07 70 07 f8 07 84 08
0190   9c 08 74 08 00 08 cc 05 2c 01 78 fc 18 fa 44 fa
01a0   78 fb 34 fc f8 fb 5c fb 0c fb 10 fb 40 fb 5c fb
01b0   60 fb 4c fb 2c fb 20 fb 34 fb 50 fb 4c fb 40 fb
01c0   40 fb 40 fb 40 fb 44 fb 48 fb 44 fb 40 fb 48 fb
01d0   4c fb 38 fb 2c fb 44 fb 40 fb 4c fb 38 fb 28 fb
01e0   24 fb 30 fb 54 fb 48 fb 4c fb 30 fb e4 00 80 d6
01f0   00 00 cc 93 60 6b d9 0c 03 00 00 00 09 5a 50 63
0200   e2 d1 31 81 14 00 00 00 8a 20 00 20 02 20 00 30
0210   8a 20 00 21 02 20 00 31 00 00 00 70 b8 00 00 10
0220   00 00 b4 00 24 fb 28 fb 2c fb 3c fb 38 fb 3c fb
0230   2c fb 2c fb 1c fb 28 fb 3c fb 34 fb 38 fb 34 fb
0240   28 fb 24 fb 24 fb 38 fb 38 fb 2c fb 34 fb 34 fb
0250   34 fb 2c fb 30 fb 28 fb 34 fb 2c fb 24 fb 34 fb
0260   2c fb 38 fb c4 fb d0 fe f0 03 24 08 b8 09 00 09
0270   b8 07 58 07 cc 07 68 08 a0 08 6c 08 20 08 60 06
0280   38 02 44 fd 44 fa 14 fa 34 fb 14 fc 10 fc 68 fb
0290   00 fb 00 fb 38 fb 64 fb 5c fb 3c fb 24 fb 30 fb
02a0   44 fb 44 fb 48 fb 34 fb 2c fb 3c fb 40 fb 54 fb
02b0   44 fb 28 fb 30 fb 34 fb 34 fb 30 fb 44 fb 3c fb
02c0   38 fb 38 fb 40 fb 3c fb 2c fb 24 fb 34 fb 40 fb
02d0   48 fb 38 fb 28 fb 30 fb e4 00 80 d6 00 00 cd 93
02e0   60 6b d9 0c 04 00 00 00 09 5a 50 63 e2 2a 6f 81
02f0   14 00 00 00 22 34 00 20 02 30 00 30 22 34 00 21
0300   02 30 00 31 00 00 00 70 b8 00 00 10 00 00 b4 00
0310   2c fb 1c fb 0c fb 18 fb 24 fb 28 fb 18 fb 14 fb
0320   0c fb 20 fb 24 fb 24 fb 0c fb 14 fb 30 fb 20 fb
0330   18 fb 14 fb 20 fb 18 fb 0c fb 10 fb 14 fb 18 fb
0340   08 fb fc fa 00 fb 0c fb 10 fb 18 fb 08 fb 00 fb
0350   1c fb a8 fc f0 00 04 06 14 09 50 09 2c 08 44 07
0360   54 07 e8 07 68 08 60 08 24 08 7c 07 90 04 a4 ff
0370   4c fb b4 f9 60 fa 8c fb 04 fc a4 fb f8 fa c4 fa
0380   e4 fa 28 fb 40 fb 24 fb 08 fb f8 fa f8 fa 08 fb
0390   0c fb 1c fb 10 fb 0c fb 08 fb 08 fb 10 fb 0c fb
03a0   08 fb 00 fb 20 fb 20 fb 08 fb 08 fb 14 fb 1c fb
03b0   18 fb 18 fb 24 fb 14 fb 20 fb 18 fb 10 fb 0c fb
03c0   14 fb 14 fb e4 00 80 d6 00 00 ce 93 60 6b d9 0c
03d0   05 00 00 00 09 5a 50 63 e2 83 ac 81 14 00 00 00
03e0   ba 47 00 20 02 40 00 30 ba 47 00 21 02 40 00 31
03f0   00 00 00 70 b8 00 00 10 00 00 b4 00 40 fb 3c fb
0400   44 fb 4c fb 44 fb 44 fb 44 fb 3c fb 50 fb 50 fb
0410   44 fb 3c fb 50 fb 40 fb 44 fb 40 fb 4c fb 50 fb
0420   4c fb 3c fb 40 fb 48 fb 50 fb 4c fb 54 fb 48 fb
0430   40 fb 40 fb 48 fb 4c fb 40 fb 48 fb 4c fb 88 fb
0440   90 fd 1c 02 fc 06 a0 09 74 09 3c 08 88 07 b4 07
0450   50 08 bc 08 a8 08 5c 08 4c 07 00 04 f8 fe 20 fb
0460   f4 f9 d8 fa dc fb 2c fc c4 fb 30 fb fc fa 1c fb
0470   58 fb 78 fb 64 fb 44 fb 30 fb 34 fb 4c fb 44 fb
0480   48 fb 38 fb 50 fb 40 fb 44 fb 54 fb 54 fb 3c fb
0490   3c fb 48 fb 50 fb 54 fb 50 fb 48 fb 48 fb 48 fb
04a0   50 fb 44 fb 5c fb 58 fb 40 fb 44 fb 44 fb 50 fb
04b0   49 62 20 12
*/

/*
========
MLink
54 53 50 2a 67 5c 2d 01 01 00 00 00
========
MStream fragment
len = 232 [0:2] length in bytes of fragment payload (not including fragment header, including MStream payload header)
e8 00
flags + subtype [2]
80
device id [3]
d6
fragment offset [4:6]
00 00
fragment id [6:8]
ca 93
========
Fragment payload
========
MStream payload header
========
device serial [0:4]
60 6b d9 0c
event num [4:7]
01 00 00
unused [7]
00
========
MStream payload
========
[8:232]
0010                                       09 5a 50 63
0020   82 c6 79 80 18 00 00 00 c1 05 00 20 4a 2b 00 40
0030   03 00 00 30 c1 05 00 21 02 00 00 31 00 00 00 70
0040   b8 00 00 10 00 00 b4 00 40 fb 34 fb 34 fb 38 fb
0050   40 fb 48 fb 38 fb 34 fb 34 fb 40 fb 3c fb 50 fb
0060   40 fb 44 fb 2c fb 34 fb 44 fb 34 fb 40 fb 48 fb
0070   3c fb 48 fb 44 fb 3c fb 3c fb 4c fb 44 fb 40 fb
0080   40 fb 38 fb 3c fb 38 fb e4 fb ec fe f8 03 40 08
0090   c0 09 fc 08 d8 07 74 07 e4 07 7c 08 b8 08 90 08
00a0   28 08 68 06 24 02 38 fd 4c fa 24 fa 44 fb 14 fc
00b0   0c fc 7c fb 08 fb 04 fb 38 fb 60 fb 60 fb 4c fb
00c0   34 fb 2c fb 38 fb 38 fb 48 fb 3c fb 30 fb 3c fb
00d0   3c fb 50 fb 34 fb 40 fb 3c fb 34 fb 2c fb 34 fb
00e0   40 fb 40 fb 38 fb 40 fb 30 fb 2c fb 28 fb 30 fb
00f0   34 fb 34 fb 30 fb 24 fb 24 fb 34 fb
========
...
*/
