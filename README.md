# smpp-test-harness

## Overview

This repository contains a series of tools for testing SMPP 3.4 architectures, as
well as methods to simplify the construction of customer SMPP test tools.  It
is written in golang and depends on the smpp-go library.  On top of that library,
it adds a simplistic ESME (client) and SMSC (server) implementation, as well
as a method for launching and controlling multiple ESME and/or SMSC instances.
Furthermore, it adds a terminal-based UI for interacting with the launched
ESME and SMSC applications.

## Structure

smpp-go provides methods for encoding and decoding SMPP PDUs.  smpp-test-harness
adds SMPP Agents, which can operate as an ESME (meaning it initiates the outbound
flow toward one or more peers) or an SMSC (meaning it listens for inbound
flows from one or more peers).  An AgentGroup instance is responsible for
launching a group of agents.  The AgentGroup provides a sender channel and is
provided an event channel.  The sender channel receives messages that specify a
PDU for an Agent to send, the Agent which should send it, and the peer to which the
Agent should try to send the PDU.  When specific events happen on an Agent,
the Agent sends a message on the event channel describing that event.  Events include:

* Receipt of a PDU from a peer
* Delivery of a PDU to a peer
* Delivery of a bind to a peer
* Receipt of a bind from a peer
* Completion of a bind on a transport

The configuration of the Agents in an AgentGroup can be defined in a YAML file.
A YAMLConfigReader can be used to read the YAML file, with the resulting objects
supplied to the AgentGroup.

A TextInteractionBroker instance can read text instructions from a line-oriented
io.Reader, and output human-consumable information to a line-oriented io.Writer.
The command set includes:

* **help** - show set of supported commands
* **&lt;source_peer&gt;: send &lt;PDU_type&gt; to &lt;peer_name&gt; [&lt;params&gt;]** - send a PDU from a member of a the AgentGroup to one of its peer, by name.  This includes the following &lt;PDU_type&gt;s:
  * `&lt;source_peer&gt;: send enquire-link to &lt;peer_name&gt;` - send enquire-link to peer
  * `&lt;source_peer&gt;: send submit-sm to &lt;peer_name&gt; dest_addr=&lt;dest_addr_string&gt; short_message=&lt;message_string&gt;` - send a submit-sm.  If &lt;dest_addr_string&gt; is provided, set the dest_addr field to this value (up to the limit of the field size). Otherwise, the field value is null.  If &lt;message_string&gt; is provided, set the short_message field to this value (up to the field limit value) and set the sm_length field to match. If this value is not provided, a default message is used.

In the app/ directory, smpp-test-agent.go is an application that can be used to read a YAML
definition and launch a group of ESMEs or SMSCs.  It is started as follows:

```bash
smpp-test-agent run esmes|smscs <path/to/config.yaml>
```

`esmes` or `smscs` instructs the smpp-test-agent to start either the ESMEs specified in the
YAML config (and initiate the declared outbound binds) or the SMSCs specified in
the YAML config.
