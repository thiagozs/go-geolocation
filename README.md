# Simple GeoLocation API

Geolocation based on ip address.

HealthCheck through `"/healthz"` and `"/readiness"` endpoints, status code are 200 response.

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

## Versioning and license

Our version numbers follow the [semantic versioning specification](http://semver.org/). You can see the available versions by checking the [tags on this repository](https://github.com/mercadobitcoin/go-geolocation/tags). For more details about our license model, please take a look at the [LICENSE](LICENSE) file.

**2023**, thiagozs.