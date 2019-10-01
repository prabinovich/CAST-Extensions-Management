package main

import "fmt"

func VersionOrdinal(version string) string {
    // ISO/IEC 14651:2011
    const maxByte = 1<<8 - 1
    vo := make([]byte, 0, len(version)+8)
    j := -1
    for i := 0; i < len(version); i++ {
        b := version[i]
        if '0' > b || b > '9' {
            vo = append(vo, b)
            j = -1
            continue
        }
        if j == -1 {
            vo = append(vo, 0x00)
            j = len(vo) - 1
        }
        if vo[j] == 1 && vo[j+1] == '0' {
            vo[j+1] = b
            continue
        }
        if vo[j]+1 > maxByte {
            panic("VersionOrdinal: invalid version")
        }
        vo = append(vo, b)
        vo[j]++
    }
    return string(vo)
}

func main() {
    versions := []struct{ a, b string }{
        {"1.05.00.0156", "1.0.221.9289"},
        // Go versions
        {"1.0.9", "1.0.10"},
    }
    for _, version := range versions {
        a, b := VersionOrdinal(version.a), VersionOrdinal(version.b)
        switch {
        case a > b:
            fmt.Println(version.a, ">", version.b)
        case a < b:
            fmt.Println(version.a, "<", version.b)
        case a == b:
            fmt.Println(version.a, "=", version.b)
        }
    }
}