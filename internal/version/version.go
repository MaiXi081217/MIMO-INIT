package version

import (
    "strconv"
    "strings"
)

func versionLess(v1, v2 string) bool {
    v1 = strings.TrimPrefix(v1, "V")
    v2 = strings.TrimPrefix(v2, "V")

    a1, _ := strconv.Atoi(strings.Split(v1, ".")[0])
    b1, _ := strconv.Atoi(strings.Split(v1, ".")[1])
    a2, _ := strconv.Atoi(strings.Split(v2, ".")[0])
    b2, _ := strconv.Atoi(strings.Split(v2, ".")[1])

    return a1 < a2 || (a1 == a2 && b1 < b2)
}
