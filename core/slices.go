package core

func ClearBuffer(b []byte) {
    for i := range b {
        b[i] = 0
    }
}

func Concat(s1 []int64, s2 []int) []int64 {
    sz := len(s1)
    res := make([]int64, sz+len(s2))
    copy(res, s1)

    for i, v := range s2 {
        res[sz+i] = int64(v)
    }

    return res
}
