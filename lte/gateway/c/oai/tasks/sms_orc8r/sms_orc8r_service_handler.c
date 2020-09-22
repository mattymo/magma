/*
 * Licensed to the OpenAirInterface (OAI) Software Alliance under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The OpenAirInterface Software Alliance licenses this file to You under
 * the terms found in the LICENSE file in the root of this source tree.
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *-------------------------------------------------------------------------------
 * For more information about the OpenAirInterface (OAI) Software Alliance:
 *      contact@openairinterface.org
 */

#include <string.h>

#include "assertions.h"
#include "intertask_interface.h"
#include "log.h"
#include "common_defs.h"
#include "intertask_interface_types.h"
#include "itti_types.h"
#include "sms_orc8r_messages.h"
#include "sgs_messages_types.h"

int handle_sms_orc8r_downlink_unitdata(
    const itti_sgsap_downlink_unitdata_t* sgs_dl_unitdata_p) {
  int rc = RETURNok;

  MessageDef* message_p                              = NULL;
  itti_sgsap_downlink_unitdata_t* sgs_dl_unit_data_p = NULL;

  message_p = itti_alloc_new_message(TASK_SMS_ORC8R, SGSAP_DOWNLINK_UNITDATA);
  AssertFatal(message_p, "itti_alloc_new_message Failed");
  sgs_dl_unit_data_p = &message_p->ittiMsg.sgsap_downlink_unitdata;
  memset((void*) sgs_dl_unit_data_p, 0, sizeof(itti_sgsap_downlink_unitdata_t));
  OAILOG_DEBUG(LOG_SMS_ORC8R, "Received SMO Downlink UnitData message from Orc8r\n");
  memcpy(
      sgs_dl_unit_data_p, sgs_dl_unitdata_p,
      sizeof(itti_sgsap_downlink_unitdata_t));
  // send it to NAS module for further processing
  rc = send_msg_to_task(&sms_orc8r_task_zmq_ctx, TASK_MME_APP, message_p);
  OAILOG_FUNC_RETURN(LOG_NAS_EMM, rc);
}

int handle_sms_orc8r_paging_request(
    const itti_sgsap_paging_request_t* const sgs_paging_req_pP) {
  MessageDef* message_p = NULL;
  int rc                = RETURNok;
  OAILOG_FUNC_IN(LOG_SMS_ORC8R);

  /* Received Paging Request from orc8r, send it to MME App for further
   * processing.
   */
  message_p = itti_alloc_new_message(TASK_SMS_ORC8R, SGSAP_PAGING_REQUEST);
  AssertFatal(
      message_p,
      "itti_alloc_new_message Failed while handling Paging Request from "
      "orc8r");

  itti_sgsap_paging_request_t* sgs_paging_req_p =
      &message_p->ittiMsg.sgsap_paging_request;
  memset((void*) sgs_paging_req_p, 0, sizeof(itti_sgsap_paging_request_t));

  memcpy(
      (void*) sgs_paging_req_p, (void*) sgs_paging_req_pP,
      sizeof(itti_sgsap_paging_request_t));

  OAILOG_DEBUG(
      LOG_SMS_ORC8R,
      "Received Paging Request message from orc8r and send Paging request "
      "to "
      "MME app"
      "for Imsi :%s \n",
      sgs_paging_req_p->imsi);
  rc = send_msg_to_task(&sms_orc8r_task_zmq_ctx, TASK_MME_APP, message_p);

  OAILOG_FUNC_RETURN(LOG_NAS_EMM, rc);
}
