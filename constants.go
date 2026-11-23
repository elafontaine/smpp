package smpp

var commandIdByHex = map[string]map[string]string{
	"80000000": {"hex": "80000000", "name": "generic_nack"},
	"00000001": {"hex": "00000001", "name": "bind_receiver"},
	"80000001": {"hex": "80000001", "name": "bind_receiver_resp"},
	"00000002": {"hex": "00000002", "name": "bind_transmitter"},
	"80000002": {"hex": "80000002", "name": "bind_transmitter_resp"},
	"00000003": {"hex": "00000003", "name": "query_sm"},
	"80000003": {"hex": "80000003", "name": "query_sm_resp"},
	"00000004": {"hex": "00000004", "name": "submit_sm"},
	"80000004": {"hex": "80000004", "name": "submit_sm_resp"},
	"00000005": {"hex": "00000005", "name": "deliver_sm"},
	"80000005": {"hex": "80000005", "name": "deliver_sm_resp"},
	"00000006": {"hex": "00000006", "name": "unbind"},
	"80000006": {"hex": "80000006", "name": "unbind_resp"},
	"00000007": {"hex": "00000007", "name": "replace_sm"},
	"80000007": {"hex": "80000007", "name": "replace_sm_resp"},
	"00000008": {"hex": "00000008", "name": "cancel_sm"},
	"80000008": {"hex": "80000008", "name": "cancel_sm_resp"},
	"00000009": {"hex": "00000009", "name": "bind_transceiver"},
	"80000009": {"hex": "80000009", "name": "bind_transceiver_resp"},
	"0000000b": {"hex": "0000000b", "name": "outbind"},
	"00000015": {"hex": "00000015", "name": "enquire_link"},
	"80000015": {"hex": "80000015", "name": "enquire_link_resp"},
	"00000021": {"hex": "00000021", "name": "submit_multi"},
	"80000021": {"hex": "80000021", "name": "submit_multi_resp"},
	"00000102": {"hex": "00000102", "name": "alert_notification"},
	"00000103": {"hex": "00000103", "name": "data_sm"},
	"80000103": {"hex": "80000103", "name": "data_sm_resp"},

	// v4 codes

	"80010000": {"hex": "80010000", "name": "generic_nack_v4"},
	"00010001": {"hex": "00010001", "name": "bind_receiver_v4"},
	"80010001": {"hex": "80010001", "name": "bind_receiver_resp_v4"},
	"00010002": {"hex": "00010002", "name": "bind_transmitter_v4"},
	"80010002": {"hex": "80010002", "name": "bind_transmitter_resp_v4"},
	"00010003": {"hex": "00010003", "name": "query_sm_v4"},
	"80010003": {"hex": "80010003", "name": "query_sm_resp_v4"},
	"00010004": {"hex": "00010004", "name": "submit_sm_v4"},
	"80010004": {"hex": "80010004", "name": "submit_sm_resp_v4"},
	"00010005": {"hex": "00010005", "name": "deliver_sm_v4"},
	"80010005": {"hex": "80010005", "name": "deliver_sm_resp_v4"},
	"00010006": {"hex": "00010006", "name": "unbind_v4"},
	"80010006": {"hex": "80010006", "name": "unbind_resp_v4"},
	"00010007": {"hex": "00010007", "name": "replace_sm_v4"},
	"80010007": {"hex": "80010007", "name": "replace_sm_resp_v4"},
	"00010008": {"hex": "00010008", "name": "cancel_sm_v4"},
	"80010008": {"hex": "80010008", "name": "cancel_sm_resp_v4"},
	"00010009": {"hex": "00010009", "name": "delivery_receipt_v4"},
	"80010009": {"hex": "80010009", "name": "delivery_receipt_resp_v4"},
	"0001000a": {"hex": "0001000a", "name": "enquire_link_v4"},
	"8001000a": {"hex": "8001000a", "name": "enquire_link_resp_v4"},
	"0001000b": {"hex": "0001000b", "name": "outbind_v4"},
}

var commandStatusByHex = map[string]map[string]string{
	"00000000": {"hex": "00000000", "name": ESME_ROK, "description": "No error"},
	"00000001": {"hex": "00000001", "name": ESME_RINVMSGLEN, "description": "Message Length is invalid"},
	"00000002": {"hex": "00000002", "name": ESME_RINVCMDLEN, "description": "Command Length is invalid"},
	"00000003": {"hex": "00000003", "name": ESME_RINVCMDID, "description": "Invalid Command ID"},
	"00000004": {"hex": "00000004", "name": ESME_RINVBNDSTS, "description": "Incorrect BIND Status for given command"},
	"00000005": {"hex": "00000005", "name": ESME_RALYBND, "description": "ESME Already in bound state"},
	"00000006": {"hex": "00000006", "name": ESME_RINVPRTFLG, "description": "Invalid priority flag"},
	"00000007": {"hex": "00000007", "name": ESME_RINVREGDLVFLG, "description": "Invalid registered delivery flag"},
	"00000008": {"hex": "00000008", "name": ESME_RSYSERR, "description": "System Error"},
	"0000000a": {"hex": "0000000a", "name": ESME_RINVSRCADR, "description": "Invalid source address"},
	"0000000b": {"hex": "0000000b", "name": ESME_RINVDSTADR, "description": "Invalid destination address"},
	"0000000c": {"hex": "0000000c", "name": ESME_RINVMSGID, "description": "Message ID is invalid"},
	"0000000d": {"hex": "0000000d", "name": ESME_RBINDFAIL, "description": "Bind failed"},
	"0000000e": {"hex": "0000000e", "name": ESME_RINVPASWD, "description": "Invalid password"},
	"0000000f": {"hex": "0000000f", "name": ESME_RINVSYSID, "description": "Invalid System ID"},
	"00000011": {"hex": "00000011", "name": ESME_RCANCELFAIL, "description": "Cancel SM Failed"},
	"00000013": {"hex": "00000013", "name": ESME_RREPLACEFAIL, "description": "Replace SM Failed"},
	"00000014": {"hex": "00000014", "name": ESME_RMSGQFUL, "description": "Message queue full"},
	"00000015": {"hex": "00000015", "name": ESME_RINVSERTYP, "description": "Invalid service type"},
	"00000033": {"hex": "00000033", "name": ESME_RINVNUMDESTS, "description": "Invalid number of destinations"},
	"00000034": {"hex": "00000034", "name": ESME_RINVDLNAME, "description": "Invalid distribution list name"},
	"00000040": {"hex": "00000040", "name": ESME_RINVDESTFLAG, "description": "Destination flag is invalid (submit_multi)"},
	"00000042": {"hex": "00000042", "name": ESME_RINVSUBREP, "description": "Invalid `submit with replace` request (i.e. submit_sm with replace_if_present_flag set)"},
	"00000043": {"hex": "00000043", "name": ESME_RINVESMCLASS, "description": "Invalid esm_class field data"},
	"00000044": {"hex": "00000044", "name": ESME_RCNTSUBDL, "description": "Cannot submit to distribution list"},
	"00000045": {"hex": "00000045", "name": ESME_RSUBMITFAIL, "description": "submit_sm or submit_multi failed"},
	"00000048": {"hex": "00000048", "name": ESME_RINVSRCTON, "description": "Invalid source address TON"},
	"00000049": {"hex": "00000049", "name": ESME_RINVSRCNPI, "description": "Invalid source address NPI"},
	"00000050": {"hex": "00000050", "name": ESME_RINVDSTTON, "description": "Invalid destination address TON"},
	"00000051": {"hex": "00000051", "name": ESME_RINVDSTNPI, "description": "Invalid destination address NPI"},
	"00000053": {"hex": "00000053", "name": ESME_RINVSYSTYP, "description": "Invalid system_type field"},
	"00000054": {"hex": "00000054", "name": ESME_RINVREPFLAG, "description": "Invalid replace_if_present flag"},
	"00000055": {"hex": "00000055", "name": ESME_RINVNUMMSGS, "description": "Invalid number of messages"},
	"00000058": {"hex": "00000058", "name": ESME_RTHROTTLED, "description": "Throttling error (ESME has exceeded allowed message limits)"},
	"00000061": {"hex": "00000061", "name": ESME_RINVSCHED, "description": "Invalid scheduled delivery time"},
	"00000062": {"hex": "00000062", "name": ESME_RINVEXPIRY, "description": "Invalid message validity period (expiry time)"},
	"00000063": {"hex": "00000063", "name": ESME_RINVDFTMSGID, "description": "Predefined message invalid or not found"},
	"00000064": {"hex": "00000064", "name": ESME_RX_T_APPN, "description": "ESME Receiver Temporary App Error Code"},
	"00000065": {"hex": "00000065", "name": ESME_RX_P_APPN, "description": "ESME Receiver Permanent App Error Code"},
	"00000066": {"hex": "00000066", "name": ESME_RX_R_APPN, "description": "ESME Receiver Reject Message Error Code"},
	"00000067": {"hex": "00000067", "name": ESME_RQUERYFAIL, "description": "query_sm request failed"},
	"000000c0": {"hex": "000000c0", "name": ESME_RINVOPTPARSTREAM, "description": "Error in the optional part of the PDU Body"},
	"000000c1": {"hex": "000000c1", "name": ESME_ROPTPARNOTALLWD, "description": "Optional paramenter not allowed"},
	"000000c2": {"hex": "000000c2", "name": ESME_RINVPARLEN, "description": "Invalid parameter length"},
	"000000c3": {"hex": "000000c3", "name": ESME_RMISSINGOPTPARAM, "description": "Expected optional parameter missing"},
	"000000c4": {"hex": "000000c4", "name": ESME_RINVOPTPARAMVAL, "description": "Invalid optional parameter value"},
	"000000fe": {"hex": "000000fe", "name": ESME_RDELIVERYFAILURE, "description": "Delivery Failure (used for data_sm_resp)"},
	"000000ff": {"hex": "000000ff", "name": ESME_RUNKNOWNERR, "description": "Unknown error"},
}

var mandatoryParameterLists = map[string][]map[string]interface{}{
	"bind_transmitter": { // SMPP v3.4, section 4.1.1, table 4-1, page 46
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
		{"name": "password", "min": 1, "max": 9, "type": "string"},
		{"name": "system_type", "min": 1, "max": 13, "type": "string"},
		{"name": "interface_version", "min": 1, "max": 1, "type": "hex"},
		{"name": "addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "address_range", "min": 1, "max": 41, "type": "string"},
	},
	"bind_transmitter_resp": { // SMPP v3.4, section 4.1.2, table 4-2, page 47
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
	},
	"bind_receiver": { // SMPP v3.4, section 4.1.3, table 4-3, page 48
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
		{"name": "password", "min": 1, "max": 9, "type": "string"},
		{"name": "system_type", "min": 1, "max": 13, "type": "string"},
		{"name": "interface_version", "min": 1, "max": 1, "type": "hex"},
		{"name": "addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "address_range", "min": 1, "max": 41, "type": "string"},
	},
	"bind_receiver_resp": { // SMPP v3.4, section 4.1.4, table 4-4, page 50
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
	},
	"bind_transceiver": { // SMPP v3.4, section 4.1.5, table 4-5, page 51
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
		{"name": "password", "min": 1, "max": 9, "type": "string"},
		{"name": "system_type", "min": 1, "max": 13, "type": "string"},
		{"name": "interface_version", "min": 1, "max": 1, "type": "hex"},
		{"name": "addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "address_range", "min": 1, "max": 41, "type": "string"},
	},
	"bind_transceiver_resp": { // SMPP v3.4, section 4.1.6, table 4-6, page 52
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
	},
	"outbind": { // SMPP v3.4, section 4.1.7.1, page 54
		{"name": "system_id", "min": 1, "max": 16, "type": "string"},
		{"name": "password", "min": 1, "max": 9, "type": "string"},
	},
	"unbind": { // SMPP v3.4, section 4.2.1, table 4-7, page 56
	},
	"unbind_resp": { // SMPP v3.4, section 4.2.2, table 4-8, page 56
	},
	"generic_nack": { // SMPP v3.4, section 4.3.1, table 4-9, page 57
	},
	"submit_sm": { // SMPP v3.4, section 4.4.1, table 4-10, page 59-61
		{"name": "service_type", "min": 1, "max": 6, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "esm_class", "min": 1, "max": 1, "type": "integer"},
		{"name": "protocol_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "priority_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "schedule_delivery_time", "min": 1, "max": 17, "type": "string"},
		{"name": "validity_period", "min": 1, "max": 17, "type": "string"},
		{"name": "registered_delivery", "min": 1, "max": 1, "type": "integer"},
		{"name": "replace_if_present_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "data_coding", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_default_msg_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_length", "min": 1, "max": 1, "type": "integer"},
		{"name": "short_message", "min": 0, "max": 254, "type": "xstring"},
	},
	"submit_sm_resp": { // SMPP v3.4, section 4.4.2, table 4-11, page 67
		{"name": "message_id", "min": 0, "max": 65, "type": "string"},
	},
	"submit_multi": { // SMPP v3.4, section 4.5.1, table 4-12, page 69-71
		{"name": "service_type", "min": 1, "max": 6, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "number_of_dests", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_address", "min": 0, "max": 0, "type": "dest_address"},
		{"name": "esm_class", "min": 1, "max": 1, "type": "integer"},
		{"name": "protocol_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "priority_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "schedule_delivery_time", "min": 1, "max": 17, "type": "string"},
		{"name": "validity_period", "min": 1, "max": 17, "type": "string"},
		{"name": "registered_delivery", "min": 1, "max": 1, "type": "integer"},
		{"name": "replace_if_present_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "data_coding", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_default_msg_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_length", "min": 1, "max": 1, "type": "integer"},
		{"name": "short_message", "min": 0, "max": 254, "type": "xstring"},
	},
	"dest_address": { // SMPP v3.4, section 4.5.1.1, table 4-13, page 75
		{"name": "dest_flag", "min": 1, "max": 1, "type": "integer"},
		// "sme_dest_address" or "distribution_list" goes here
	},
	"sme_dest_address": { // SMPP v3.4, section 4.5.1.1, table 4-14, page 75
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 21, "type": "string"},
	},
	"distribution_list": { // SMPP v3.4, section 4.5.1.2, table 4-15, page 75
		{"name": "dl_name", "min": 1, "max": 21, "type": "string"},
	},
	"submit_multi_resp": { // SMPP v3.4, section 4.5.2, table 4-16, page 76
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
		{"name": "no_unsuccess", "min": 1, "max": 1, "type": "integer"},
		{"name": "unsuccess_sme", "min": 0, "max": 0, "type": "unsuccess_sme"},
	},
	"unsuccess_sme": { // SMPP v3.4, section 4.5.2.1, table 4-17, page 77
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "error_status_code", "min": 4, "max": 4, "type": "integer"},
	},
	"deliver_sm": { // SMPP v3.4, section 4.6.1, table 4-18, page 79-81
		{"name": "service_type", "min": 1, "max": 6, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "esm_class", "min": 1, "max": 1, "type": "integer"},
		{"name": "protocol_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "priority_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "schedule_delivery_time", "min": 1, "max": 1, "type": "string"},
		{"name": "validity_period", "min": 1, "max": 1, "type": "string"},
		{"name": "registered_delivery", "min": 1, "max": 1, "type": "integer"},
		{"name": "replace_if_present_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "data_coding", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_default_msg_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_length", "min": 1, "max": 1, "type": "integer"},
		{"name": "short_message", "min": 0, "max": 254, "type": "xstring"},
	},
	"deliver_sm_resp": { // SMPP v3.4, section 4.6.2, table 4-19, page 85
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
	},
	"data_sm": { // SMPP v3.4, section 4.7.1, table 4-20, page 87-88
		{"name": "service_type", "min": 1, "max": 6, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 65, "type": "string"},
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 65, "type": "string"},
		{"name": "esm_class", "min": 1, "max": 1, "type": "integer"},
		{"name": "registered_delivery", "min": 1, "max": 1, "type": "integer"},
		{"name": "data_coding", "min": 1, "max": 1, "type": "integer"},
	},
	"data_sm_resp": { // SMPP v3.4, section 4.7.2, table 4-21, page 93
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
	},
	"query_sm": { // SMPP v3.4, section 4.8.1, table 4-22, page 95
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
	},
	"query_sm_resp": { // SMPP v3.4, section 4.7.2, table 4-21, page 93
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
		{"name": "final_date", "min": 1, "max": 17, "type": "string"},
		{"name": "message_state", "min": 1, "max": 1, "type": "integer"},
		{"name": "error_code", "min": 1, "max": 1, "type": "integer"},
	},
	"cancel_sm": { // SMPP v3.4, section 4.9.1, table 4-24, page 98-99
		{"name": "service_type", "min": 1, "max": 6, "type": "string"},
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "dest_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "dest_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "destination_addr", "min": 1, "max": 21, "type": "string"},
	},
	"cancel_sm_resp": { // SMPP v3.4, section 4.9.2, table 4-25, page 100
	},
	"replace_sm": { // SMPP v3.4, section 4.10.1, table 4-26, page 102-103
		{"name": "message_id", "min": 1, "max": 65, "type": "string"},
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 21, "type": "string"},
		{"name": "schedule_delivery_time", "min": 1, "max": 17, "type": "string"},
		{"name": "validity_period", "min": 1, "max": 17, "type": "string"},
		{"name": "registered_delivery", "min": 1, "max": 1, "type": "integer"},
		{"name": "replace_if_present_flag", "min": 1, "max": 1, "type": "integer"},
		{"name": "data_coding", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_default_msg_id", "min": 1, "max": 1, "type": "integer"},
		{"name": "sm_length", "min": 1, "max": 1, "type": "integer"},
		{"name": "short_message", "min": 0, "max": 254, "type": "xstring"},
	},
	"replace_sm_resp": { // SMPP v3.4, section 4.10.2, table 4-27, page 104
	},
	"enquire_link": { // SMPP v3.4, section 4.11.1, table 4-28, page 106
	},
	"enquire_link_resp": { // SMPP v3.4, section 4.11.2, table 4-29, page 106
	},
	"alert_notification": { // SMPP v3.4, section 4.12.1, table 4-30, page 108
		{"name": "source_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "source_addr", "min": 1, "max": 65, "type": "string"},
		{"name": "esme_addr_ton", "min": 1, "max": 1, "type": "integer"},
		{"name": "esme_addr_npi", "min": 1, "max": 1, "type": "integer"},
		{"name": "esme_addr", "min": 1, "max": 65, "type": "string"},
	},
}
var optionalParameterTagByHex = map[string]map[string]interface{}{
	"0005": {"hex": "0005", "name": "dest_addr_subunit", "type": "integer", "tech": "GSM"},
	"0006": {"hex": "0006", "name": "dest_network_type", "type": "integer", "tech": "Generic"},
	"0007": {"hex": "0007", "name": "dest_bearer_type", "type": "integer", "tech": "Generic"},
	"0008": {"hex": "0008", "name": "dest_telematics_id", "type": "integer", "tech": "GSM", "min": 2},

	"000d": {"hex": "000d", "name": "source_addr_subunit", "type": "integer", "tech": "GSM"},
	"000e": {"hex": "000e", "name": "source_network_type", "type": "integer", "tech": "Generic"},
	"000f": {"hex": "000f", "name": "source_bearer_type", "type": "integer", "tech": "Generic"},
	"0010": {"hex": "0010", "name": "source_telematics_id", "type": "integer", "tech": "GSM"},

	"0017": {"hex": "0017", "name": "qos_time_to_live", "type": "integer", "tech": "Generic", "min": 4},
	"0019": {"hex": "0019", "name": "payload_type", "type": "integer", "tech": "Generic"},
	"001d": {"hex": "001d", "name": "additional_status_info_text", "type": "string", "tech": "Generic"},
	"001e": {"hex": "001e", "name": "receipted_message_id", "type": "string", "tech": "Generic"},

	"0030": {"hex": "0030", "name": "ms_msg_wait_facilities", "type": "bitmask", "tech": "GSM"},

	"0101": {"hex": "0101", "name": "PVCY_AuthenticationStr", "type": nil, "tech": "? (J-Phone)"},

	"0201": {"hex": "0201", "name": "privacy_indicator", "type": "integer", "tech": "CDMA, TDMA"},
	"0202": {"hex": "0202", "name": "source_subaddress", "type": "hex", "tech": "CDMA, TDMA", "min": 2},
	"0203": {"hex": "0203", "name": "dest_subaddress", "type": "hex", "tech": "CDMA, TDMA", "min": 2},
	"0204": {"hex": "0204", "name": "user_message_reference", "type": "integer", "tech": "Generic", "min": 2},
	"0205": {"hex": "0205", "name": "user_response_code", "type": "integer", "tech": "CDMA, TDMA"},

	"020a": {"hex": "020a", "name": "source_port", "type": "integer", "tech": "Generic", "min": 2},
	"020b": {"hex": "020b", "name": "destination_port", "type": "integer", "tech": "Generic", "min": 2},
	"020c": {"hex": "020c", "name": "sar_msg_ref_num", "type": "integer", "tech": "Generic", "min": 2},
	"020d": {"hex": "020d", "name": "language_indicator", "type": "integer", "tech": "CDMA, TDMA"},
	"020e": {"hex": "020e", "name": "sar_total_segments", "type": "integer", "tech": "Generic"},
	"020f": {"hex": "020f", "name": "sar_segment_seqnum", "type": "integer", "tech": "Generic"},
	"0210": {"hex": "0210", "name": "sc_interface_version", "type": "integer", "tech": "Generic"},

	"0301": {"hex": "0301", "name": "CC_CBN", "type": nil, "tech": "V4"},
	"0302": {"hex": "0302", "name": "callback_num_pres_ind", "type": "bitmask", "tech": "TDMA"},
	"0303": {"hex": "0303", "name": "callback_num_atag", "type": "hex", "tech": "TDMA"},
	"0304": {"hex": "0304", "name": "number_of_messages", "type": "integer", "tech": "CDMA"},
	"0381": {"hex": "0381", "name": "callback_num", "type": "hex", "tech": "CDMA, TDMA, GSM, iDEN", "min": 4},

	"0420": {"hex": "0420", "name": "dpf_result", "type": "integer", "tech": "Generic"},
	"0421": {"hex": "0421", "name": "set_dpf", "type": "integer", "tech": "Generic"},
	"0422": {"hex": "0422", "name": "ms_availability_status", "type": "integer", "tech": "Generic"},
	"0423": {"hex": "0423", "name": "network_error_code", "type": "hex", "tech": "Generic", "min": 3},
	"0424": {"hex": "0424", "name": "message_payload", "type": "hex", "tech": "Generic"},
	"0425": {"hex": "0425", "name": "delivery_failure_reason", "type": "integer", "tech": "Generic"},
	"0426": {"hex": "0426", "name": "more_messages_to_send", "type": "integer", "tech": "GSM"},
	"0427": {"hex": "0427", "name": "message_state", "type": "integer", "tech": "Generic"},
	"0428": {"hex": "0428", "name": "congestion_state", "type": nil, "tech": "Generic"},

	"0501": {"hex": "0501", "name": "ussd_service_op", "type": "hex", "tech": "GSM (USSD)"},

	"0600": {"hex": "0600", "name": "broadcast_channel_indicator", "type": nil, "tech": "GSM"},
	"0601": {"hex": "0601", "name": "broadcast_content_type", "type": nil, "tech": "CDMA, TDMA, GSM"},
	"0602": {"hex": "0602", "name": "broadcast_content_type_info", "type": nil, "tech": "CDMA, TDMA"},
	"0603": {"hex": "0603", "name": "broadcast_message_class", "type": nil, "tech": "GSM"},
	"0604": {"hex": "0604", "name": "broadcast_rep_num", "type": nil, "tech": "GSM"},
	"0605": {"hex": "0605", "name": "broadcast_frequency_interval", "type": nil, "tech": "CDMA, TDMA, GSM"},
	"0606": {"hex": "0606", "name": "broadcast_area_identifier", "type": nil, "tech": "CDMA, TDMA, GSM"},
	"0607": {"hex": "0607", "name": "broadcast_error_status", "type": nil, "tech": "CDMA, TDMA, GSM"},
	"0608": {"hex": "0608", "name": "broadcast_area_success", "type": nil, "tech": "GSM"},
	"0609": {"hex": "0609", "name": "broadcast_end_time", "type": nil, "tech": "CDMA, TDMA, GSM"},
	"060a": {"hex": "060a", "name": "broadcast_service_group", "type": nil, "tech": "CDMA, TDMA"},
	"060b": {"hex": "060b", "name": "billing_identification", "type": nil, "tech": "Generic"},

	"060d": {"hex": "060d", "name": "source_network_id", "type": nil, "tech": "Generic"},
	"060e": {"hex": "060e", "name": "dest_network_id", "type": nil, "tech": "Generic"},
	"060f": {"hex": "060f", "name": "source_node_id", "type": nil, "tech": "Generic"},
	"0610": {"hex": "0610", "name": "dest_node_id", "type": nil, "tech": "Generic"},
	"0611": {"hex": "0611", "name": "dest_addr_np_resolution", "type": nil, "tech": "CDMA, TDMA (US Only)"},
	"0612": {"hex": "0612", "name": "dest_addr_np_information", "type": nil, "tech": "CDMA, TDMA (US Only)"},
	"0613": {"hex": "0613", "name": "dest_addr_np_country", "type": nil, "tech": "CDMA, TDMA (US Only)"},

	"1101": {"hex": "1101", "name": "PDC_MessageClass", "type": nil, "tech": "? (J-Phone)"},
	"1102": {"hex": "1102", "name": "PDC_PresentationOption", "type": nil, "tech": "? (J-Phone)"},
	"1103": {"hex": "1103", "name": "PDC_AlertMechanism", "type": nil, "tech": "? (J-Phone)"},
	"1104": {"hex": "1104", "name": "PDC_Teleservice", "type": nil, "tech": "? (J-Phone)"},
	"1105": {"hex": "1105", "name": "PDC_MultiPartMessage", "type": nil, "tech": "? (J-Phone)"},
	"1106": {"hex": "1106", "name": "PDC_PredefinedMsg", "type": nil, "tech": "? (J-Phone)"},

	"1201": {"hex": "1201", "name": "display_time", "type": "integer", "tech": "CDMA, TDMA"},

	"1203": {"hex": "1203", "name": "sms_signal", "type": "integer", "tech": "TDMA", "min": 2},
	"1204": {"hex": "1204", "name": "ms_validity", "type": "integer", "tech": "CDMA, TDMA"},

	"1304": {"hex": "1304", "name": "IS95A_AlertOnDelivery", "type": nil, "tech": "CDMA"},
	"1306": {"hex": "1306", "name": "IS95A_LanguageIndicator", "type": nil, "tech": "CDMA"},

	"130c": {"hex": "130c", "name": "alert_on_message_delivery", "type": nil, "tech": "CDMA"},

	"1380": {"hex": "1380", "name": "its_reply_type", "type": "integer", "tech": "CDMA"},
	"1383": {"hex": "1383", "name": "its_session_info", "type": "hex", "tech": "CDMA", "min": 2},

	"1402": {"hex": "1402", "name": "operator_id", "type": nil, "tech": "vendor extension"},
	"1403": {"hex": "1403", "name": "tariff", "type": nil, "tech": "Mobile Network Code vendor extension"},
	"1450": {"hex": "1450", "name": "mcc", "type": nil, "tech": "Mobile Country Code vendor extension"},
	"1451": {"hex": "1451", "name": "mnc", "type": nil, "tech": "Mobile Network Code vendor extension"},
}

var commandIdByName = byName(commandIdByHex)
var commandStatusByName = byName(commandStatusByHex)
var optionalParameterTagByName = optionalParametersByName(optionalParameterTagByHex)

// byName indexes a ...ByHex table by each entry's "name".
func byName(byHex map[string]map[string]string) map[string]map[string]string {
	table := map[string]map[string]string{}
	for _, entry := range byHex {
		table[entry["name"]] = entry
	}
	return table
}

// optionalParametersByName indexes optionalParameterTagByHex by each entry's "name".
func optionalParametersByName(byHex map[string]map[string]interface{}) map[string]map[string]interface{} {
	table := map[string]map[string]interface{}{}
	for _, entry := range byHex {
		table[entry["name"].(string)] = entry
	}
	return table
}

/*
SMPP Status Code Variables

*/

const ESME_ROK = "ESME_ROK"
const ESME_RINVMSGLEN = "ESME_RINVMSGLEN"
const ESME_RINVCMDLEN = "ESME_RINVCMDLEN"
const ESME_RINVCMDID = "ESME_RINVCMDID"
const ESME_RINVBNDSTS = "ESME_RINVBNDSTS"
const ESME_RALYBND = "ESME_RALYBND"
const ESME_RINVPRTFLG = "ESME_RINVPRTFLG"
const ESME_RINVREGDLVFLG = "ESME_RINVREGDLVFLG"
const ESME_RSYSERR = "ESME_RSYSERR"
const ESME_RINVSRCADR = "ESME_RINVSRCADR"
const ESME_RINVDSTADR = "ESME_RINVDSTADR"
const ESME_RINVMSGID = "ESME_RINVMSGID"
const ESME_RBINDFAIL = "ESME_RBINDFAIL"
const ESME_RINVPASWD = "ESME_RINVPASWD"
const ESME_RINVSYSID = "ESME_RINVSYSID"
const ESME_RCANCELFAIL = "ESME_RCANCELFAIL"
const ESME_RREPLACEFAIL = "ESME_RREPLACEFAIL"
const ESME_RMSGQFUL = "ESME_RMSGQFUL"
const ESME_RINVSERTYP = "ESME_RINVSERTYP"
const ESME_RINVNUMDESTS = "ESME_RINVNUMDESTS"
const ESME_RINVDLNAME = "ESME_RINVDLNAME"
const ESME_RINVDESTFLAG = "ESME_RINVDESTFLAG"
const ESME_RINVSUBREP = "ESME_RINVSUBREP"
const ESME_RINVESMCLASS = "ESME_RINVESMCLASS"
const ESME_RCNTSUBDL = "ESME_RCNTSUBDL"
const ESME_RSUBMITFAIL = "ESME_RSUBMITFAIL"
const ESME_RINVSRCTON = "ESME_RINVSRCTON"
const ESME_RINVSRCNPI = "ESME_RINVSRCNPI"
const ESME_RINVDSTTON = "ESME_RINVDSTTON"
const ESME_RINVDSTNPI = "ESME_RINVDSTNPI"
const ESME_RINVSYSTYP = "ESME_RINVSYSTYP"
const ESME_RINVREPFLAG = "ESME_RINVREPFLAG"
const ESME_RINVNUMMSGS = "ESME_RINVNUMMSGS"
const ESME_RTHROTTLED = "ESME_RTHROTTLED"
const ESME_RINVSCHED = "ESME_RINVSCHED"
const ESME_RINVEXPIRY = "ESME_RINVEXPIRY"
const ESME_RINVDFTMSGID = "ESME_RINVDFTMSGID"
const ESME_RX_T_APPN = "ESME_RX_T_APPN"
const ESME_RX_P_APPN = "ESME_RX_P_APPN"
const ESME_RX_R_APPN = "ESME_RX_R_APPN"
const ESME_RQUERYFAIL = "ESME_RQUERYFAIL"
const ESME_RINVOPTPARSTREAM = "ESME_RINVOPTPARSTREAM"
const ESME_ROPTPARNOTALLWD = "ESME_ROPTPARNOTALLWD"
const ESME_RINVPARLEN = "ESME_RINVPARLEN"
const ESME_RMISSINGOPTPARAM = "ESME_RMISSINGOPTPARAM"
const ESME_RINVOPTPARAMVAL = "ESME_RINVOPTPARAMVAL"
const ESME_RDELIVERYFAILURE = "ESME_RDELIVERYFAILURE"
const ESME_RUNKNOWNERR = "ESME_RUNKNOWNERR"
