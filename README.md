# Simple GeoLocation API

Geo location based on ip address.

Check progress through /health endpoint and if IP's is equal to 200 it is ready

## Example Request

Call GET `https://server/ip?address=4.4.4.4`
  
## Example Response

```json
{
   "data":{
      "Country":{
         "IsInEuropeanUnion":false,
         "ISOCode":"US"
      },
      "City":{
         "Names":{
            "de":"Nashville",
            "en":"Nashville",
            "es":"Nashville",
            "fr":"Nashville",
            "ja":"ナッシュビル",
            "pt-BR":"Nashville",
            "ru":"Нашвилл",
            "zh-CN":"纳什维尔"
         }
      },
      "Location":{
         "AccuracyRadius":500,
         "Latitude":36.0964,
         "Longitude":-86.8212,
         "MetroCode":659,
         "TimeZone":"America/Chicago"
      },
      "Postal":{
         "Code":"37215"
      },
      "Traits":{
         "IsAnonymousProxy":false,
         "IsSatelliteProvider":false
      },
      "IP":"4.4.4.4"
   }
}
```
