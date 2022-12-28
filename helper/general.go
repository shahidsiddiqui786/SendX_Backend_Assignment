package helper

/* Get the minimum of two values
*/
func Min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

/* simple concatenation
*/
func GeneratePath(id string) string {
	return "file/" + id + ".html"
}

/* General Filter Function for the array.
   Based on the test function  value it would 
   filter the element from the array.
*/
func Filter[T any](ss []T, test func(T) bool) (ret []T) {
    for _, s := range ss {
        if test(s) {
            ret = append(ret, s)
        }
    }
    return
}