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

package srv

import (
	"fmt"
)

// ErrGetAddr returned when we can not get the address and port of the devices that sent a packet
type ErrGetAddr struct {}

func (e ErrGetAddr) Error() string {
	return fmt.Sprintf("Error while getting peer address and port")
}
