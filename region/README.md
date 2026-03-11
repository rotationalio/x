# Region

Region enumerates the clouds and regions that are possible to deploy a Rotational service to in order to ensure region identification and serialization is as small a data type as possible. Region codes are generally broken into parts: the first digit represents the cloud, e.g. a region code that starts with 1 is Linode. The second series of three digits represents the country, e.g. USA is 840 in the ISO 3166 standard. The three digits represents the zone of the datacenter, and is usually cloud specific.

## Region-Info

Generally speaking, region information can be loaded from environment variables and those environment variables are defined in a configuration. So including region information in your application is generally as simple as:

```sh
$ export REGION_INFO_ID=2840291
```

```go
type Config struct {
    ...
    Region region.Info
}

conf = Config{}
confire.Process("", &conf)

fmt.Println(conf.Region.ID.String())
// gcp-us-east-1b
```

The config acts as an entry point to the `Info` struct and to the `Country` details for even greater detail about the region. For example:

```go
fmt.Println(conf.Region.Country().CurrencyCode)
// USD
```

The `region.Info.Country()` method returns the `country.Country` data from the `go.rtnl.ai/x/country` package. This includes short, long, and unofficial country names, currency and measurement info, languages spoken, etc.

## Other Environment Variables

While the Region code is sufficient to determine name, country, zone, and cloud, you may also want to set the `$REGION_INFO_CLUSTER` environment variable to the name of the Kubernetes cluster the pod is running in (e.g. `rotational-gke-usa-1`).

A comprehensive list of environment variables is below. You can override the computed values for name, country, zone, and cloud by setting those respective environment variables.

- `$REGION_INFO_ID`: must either be set or empty; can either be the variable string (e.g. `gcp-us-east-1b`) or the integer id (e.g. `2840291`).
- `$REGION_INFO_CLUSTER`: the name of the kubernetes cluster or computing environment (e.g. `rotational-gke-usa-1`).
- `$REGION_INFO_NAME`: override the name of the region, e.g. `us-central1` for GCP instead of `gcp-us-central-1a`.
- `$REGION_INFO_COUNTRY`: override the country of the region setting the ISO 3166-1 alpha 2 code.
- `$REGION_INFO_ZONE`: override the zone of the region.
- `$REGION_INFO_CLOUD`: override the name of the cloud service provider, e.g. `Linode` instead of `LKE`.
