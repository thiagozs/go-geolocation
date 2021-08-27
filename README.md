![Raithe](https://github.com/catmullet/Simple-MaxMind-GeoLocation/blob/master/smpgeoloc.png?raw=true)
# Simple-MaxMind-GeoLocation
Geo location based on ip address.  Simple one file Geo location Golang API

## Steps to get going
1. Download application.go
2. Zip up application.go
3. Upload to aws Elastic Beanstalk on Go platform (Minimum of 2gb of memory)

Check progress through /health endpoint and if IP's is equal to 221 it is ready

## Update IP's

`<Your Server DNS>/update`

## Example Request
`<Your Server DNS>/ip?address=198.60.227.255`
  
## Example Response 
`{
  iso_code: "US",
  country_name: "United States",
  subdivision: "Idaho",
  city_name: "Idaho Falls",
  time_zone: "America/Boise"
}`
