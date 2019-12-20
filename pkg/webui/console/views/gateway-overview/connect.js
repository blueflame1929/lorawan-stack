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

import { connect } from 'react-redux'

import { getCollaboratorsList } from '../../store/actions/collaborators'
import { getApiKeysList } from '../../store/actions/api-keys'
import { selectSelectedGateway, selectSelectedGatewayId } from '../../store/selectors/gateways'
import { selectApiKeysTotalCount, selectApiKeysFetching } from '../../store/selectors/api-keys'
import {
  selectCollaboratorsTotalCount,
  selectCollaboratorsFetching,
} from '../../store/selectors/collaborators'

const mapStateToProps = state => {
  const gtwId = selectSelectedGatewayId(state)
  const collaboratorsTotalCount = selectCollaboratorsTotalCount(state, { id: gtwId })
  const apiKeysTotalCount = selectApiKeysTotalCount(state)

  return {
    gtwId,
    gateway: selectSelectedGateway(state),
    collaboratorsTotalCount,
    apiKeysTotalCount,
    statusBarFetching:
      collaboratorsTotalCount === undefined ||
      apiKeysTotalCount === undefined ||
      selectCollaboratorsFetching(state) ||
      selectApiKeysFetching(state),
  }
}
const mapDispatchToProps = dispatch => ({
  loadData(gtwId) {
    dispatch(getCollaboratorsList('gateway', gtwId))
    dispatch(getApiKeysList('gateway', gtwId))
  },
})

export default GatewayOverview =>
  connect(
    mapStateToProps,
    mapDispatchToProps,
  )(GatewayOverview)
