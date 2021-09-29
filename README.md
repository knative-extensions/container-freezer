# container-freezer


|     |     |
| --- | --- |
| STATUS | Alpha |
| Sponsoring WG | [Autoscaling](https://github.com/knative/community/blob/main/working-groups/WORKING-GROUPS.md#scaling)|

A standalone service for Knative to pause/unpause containers when request count drops to zero.

For more info, see: https://github.com/knative/serving/issues/11694

Based on: https://github.com/julz/freeze-proxy

## Installation

`ko apply -f config/`

