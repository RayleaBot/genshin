package app

import "slices"

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"os_usa", "os_euro", "os_asia", "os_cht"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region) || slices.Contains([]string{"cn_gf01", "cn_qd01"}, region)
}
