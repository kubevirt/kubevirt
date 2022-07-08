# PlatformConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NumPciSegments** | Pointer to **int32** |  | [optional] 
**IommuSegments** | Pointer to **[]int32** |  | [optional] 
**SerialNumber** | Pointer to **string** |  | [optional] 

## Methods

### NewPlatformConfig

`func NewPlatformConfig() *PlatformConfig`

NewPlatformConfig instantiates a new PlatformConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPlatformConfigWithDefaults

`func NewPlatformConfigWithDefaults() *PlatformConfig`

NewPlatformConfigWithDefaults instantiates a new PlatformConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetNumPciSegments

`func (o *PlatformConfig) GetNumPciSegments() int32`

GetNumPciSegments returns the NumPciSegments field if non-nil, zero value otherwise.

### GetNumPciSegmentsOk

`func (o *PlatformConfig) GetNumPciSegmentsOk() (*int32, bool)`

GetNumPciSegmentsOk returns a tuple with the NumPciSegments field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNumPciSegments

`func (o *PlatformConfig) SetNumPciSegments(v int32)`

SetNumPciSegments sets NumPciSegments field to given value.

### HasNumPciSegments

`func (o *PlatformConfig) HasNumPciSegments() bool`

HasNumPciSegments returns a boolean if a field has been set.

### GetIommuSegments

`func (o *PlatformConfig) GetIommuSegments() []int32`

GetIommuSegments returns the IommuSegments field if non-nil, zero value otherwise.

### GetIommuSegmentsOk

`func (o *PlatformConfig) GetIommuSegmentsOk() (*[]int32, bool)`

GetIommuSegmentsOk returns a tuple with the IommuSegments field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIommuSegments

`func (o *PlatformConfig) SetIommuSegments(v []int32)`

SetIommuSegments sets IommuSegments field to given value.

### HasIommuSegments

`func (o *PlatformConfig) HasIommuSegments() bool`

HasIommuSegments returns a boolean if a field has been set.

### GetSerialNumber

`func (o *PlatformConfig) GetSerialNumber() string`

GetSerialNumber returns the SerialNumber field if non-nil, zero value otherwise.

### GetSerialNumberOk

`func (o *PlatformConfig) GetSerialNumberOk() (*string, bool)`

GetSerialNumberOk returns a tuple with the SerialNumber field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSerialNumber

`func (o *PlatformConfig) SetSerialNumber(v string)`

SetSerialNumber sets SerialNumber field to given value.

### HasSerialNumber

`func (o *PlatformConfig) HasSerialNumber() bool`

HasSerialNumber returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


