cmake_minimum_required(VERSION 2.8)
project(GeoIP)

set(geoip_srcs
	libGeoIP/GeoIP.c
	libGeoIP/GeoIPCity.c
	libGeoIP/GeoIPUpdate.c
	libGeoIP/md5.c
	libGeoIP/regionName.c
	libGeoIP/timeZone.c)


include_directories(libGeoIP)
add_definitions(-DGEOIPDATADIR=\"\")
add_library(GeoIP STATIC ${geoip_srcs})
