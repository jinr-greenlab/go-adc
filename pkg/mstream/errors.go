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

import (
	"fmt"
)

// ErrMStreamAssemble returned when there is overlapping of fragments or a missing fragment
type ErrMStreamAssemble struct {
	What string
}

func (e ErrMStreamAssemble) Error() string {
	return fmt.Sprintf("Error while assembling MStream frame: %s", e.What)
}

type ErrMStreamTooManyFragments struct {
	Number int
}

func (e ErrMStreamTooManyFragments) Error() string {
	return fmt.Sprintf("Maximum number of MStream frame fragments is achieved: %d", e.Number)
}
