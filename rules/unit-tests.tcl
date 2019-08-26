#!/usr/bin/env tclsh

proc extract_destination_addr {smpp_buffer_var} {
    upvar $smpp_buffer_var smpp_buffer

    set  offset_in_buffer [string first \x00 $smpp_buffer 16] ;# service_type
    incr offset_in_buffer 3  ;# service_type null, server_type source_addr_ton, source_addr_npi
    set  offset_in_buffer [string first "\x00" $smpp_buffer $offset_in_buffer] ;# source_addr
    incr offset_in_buffer 3  ;# source_addr null, dest_addr_ton, dest_addr_npi
    return [string range $smpp_buffer $offset_in_buffer [string first "\x00" $smpp_buffer $offset_in_buffer]]
}

proc determine_c_octet_field_length {pdu_var field_start_index max_field_length} {
    upvar $pdu_var smpp_buffer
    binary scan $smpp_buffer "x${field_start_index}c$max_field_length" field_octets
    set field_length 1
    foreach octet $field_octets {
        if { $octet == 0 } {
            return $field_length
        }

        incr field_length
    }
    
    # PDU is invalidly formatted
    return -1
} 

proc extract_destination_addr_from_submit_sm_pdu {pdu_var} {
    upvar $pdu_var smpp_buffer
    set pdu_offset 16
    if { [set field_length [determine_c_octet_field_length smpp_buffer 16 6]] == -1 } { ;# service_type
        return ""
    }

    incr pdu_offset $field_length
    incr pdu_offset 2 ;# source_addr_ton, source_addr_npi

    if { [set field_length [determine_c_octet_field_length smpp_buffer $pdu_offset 21]] == -1 } { ;# source_addr
        return ""
    }

    incr pdu_offset $field_length    
    incr pdu_offset 2 ;# dest_addr_ton, dest_addr_npi

    if { [set field_length [determine_c_octet_field_length smpp_buffer $pdu_offset 21]] == -1 } { ;# source_addr
        return ""
    }

    binary scan $smpp_buffer "x${pdu_offset}a$field_length" dest_addr
    return [string range $dest_addr 0 end-1]  ;# remove null
}

set submit_sm ""
append submit_sm "\x00\x00\x00\x41" \
                 "\x00\x00\x00\x04" \
                 "\x00\x00\x00\x00" \
                 "\x00\x00\x00\x01" \
                 "\x00"             \
                 "\x00\x00"         \
                 "src-addr\x00"     \
                 "\x00\x00"         \
                 "dest-addr\x00"    \
                 "\x00\x00\x00"     \
                 "\x00"             \
                 "\x00"             \
                 "\x00\x00\x00\x00" \
                 "\x0c"             \
                 "test message"

set daddr [extract_destination_addr submit_sm]
if { $daddr != "dest-addr\x00" } {
    puts "extract_destination_addr for first submit_sm expect destination_addr = (dest-addr), got = ($daddr)"
}

set daddr [extract_destination_addr_from_submit_sm_pdu submit_sm]
if { $daddr != "dest-addr" } {
    puts "extract_destination_addr_from_pdu for first submit_sm expect destination_addr = (dest-addr), got = ($daddr)"
}

set field_length [determine_c_octet_field_length submit_sm 31 21]
if { $field_length != 9 } {
    puts "determine_c_octet_field_length for first submit_sm expect length = 9, got = $field_length"
}

set submit_sm [binary format H* 00000043000000040000000000000001000000000000303031313030000000000000000000001c54686973206973206120746573742073686f7274206d657373616765]
set daddr [extract_destination_addr submit_sm]
if { $daddr != "001100\x00" } {
    puts "extract_destination_addr for first submit_sm expect destination_addr = (dest-addr), got = ($daddr)"
}

set daddr [extract_destination_addr_from_submit_sm_pdu submit_sm]
if { $daddr != "001100" } {
    puts "extract_destination_addr_from_pdu for first submit_sm expect destination_addr = (001100), got = ($daddr)"
}

set field_length [determine_c_octet_field_length submit_sm 22 21]
if { $field_length != 7 } {
    puts "determine_c_octet_field_length for second submit_sm expect length = 7, got = $field_length"
}

puts "Testing Completed"
