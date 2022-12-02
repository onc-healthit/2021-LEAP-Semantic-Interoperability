package valueset

import "fmt"

type ErrMultipleRows string

func (e ErrMultipleRows) Error() string {
	return fmt.Sprintf("more than 1 row returned: %s", string(e))
}
