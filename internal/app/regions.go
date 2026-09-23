package app

import "slices"

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"os_usa", "os_euro", "os_asia", "os_cht"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region) || slices.Contains([]string{"cn_gf01", "cn_qd01"}, region)
}

// uidRegion is the region a UID's first digits name, as miao's getServer.
func uidRegion(uid string) string {
	if len(uid) > 8 {
		switch uid[:len(uid)-8] {
		case "5":
			return "cn_qd01"
		case "6":
			return "os_usa"
		case "7":
			return "os_euro"
		case "8", "18":
			return "os_asia"
		case "9":
			return "os_cht"
		}
	}
	return "cn_gf01"
}

// regionTimezone is the UTC offset a region's records are written in.
func regionTimezone(region string) int {
	switch region {
	case "os_usa":
		return -5
	case "os_euro":
		return 1
	}
	return 8
}
