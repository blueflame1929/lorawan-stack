// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { getDeviceId } from '../../../lib/selectors/id'
import {
  GET_DEV,
  GET_DEVICES_LIST_SUCCESS,
  GET_DEV_SUCCESS,
  UPDATE_DEV_SUCCESS,
} from '../actions/devices'

const defaultState = {
  entities: {},
  selectedDevice: undefined,
}

const device = function(state = {}, device) {
  return {
    ...state,
    ...device,
  }
}

const devices = function(state = defaultState, { type, payload }) {
  switch (type) {
    case GET_DEV:
      return {
        ...state,
        selectedDevice: payload.deviceId,
      }
    case UPDATE_DEV_SUCCESS:
    case GET_DEV_SUCCESS:
      const id = getDeviceId(payload)

      return {
        ...state,
        entities: {
          ...state.entities,
          [id]: device(state.entities[id], payload),
        },
      }
    case GET_DEVICES_LIST_SUCCESS:
      const entities = payload.entities.reduce(
        function(acc, dev) {
          const id = getDeviceId(dev)

          acc[id] = device(acc[id], dev)
          return acc
        },
        { ...state.entities },
      )

      return {
        ...state,
        entities,
      }
    default:
      return state
  }
}

export default devices
