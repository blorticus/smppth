# smppth

## Overview

This repository golang libraries intended to make the creating of 
SMPP test tools.

## Libraries

github.com/blorticus/smpp provides methods for encoding and decoding SMPP PDUs.  smppth
adds SMPP Agents, which can operate as an ESME (meaning it initiates the outbound
flow toward one or more peers) or an SMSC (meaning it listens for inbound
flows from one or more peers).  An AgentGroup instance is responsible for
launching a group of agents and provides a method for routing
PDUs to Agents for delivery to peers.  The AgentGroup also provides an event
channel.  When specific events happen on an Agent, the Agent sends a message on
the event channel describing that event.  Events include:

* Receipt of a PDU from a peer
* Delivery of a PDU to a peer
* Completion of a bind on a transport

The configuration of the Agents in an AgentGroup can be defined in a YAML file.
A YAMLConfigReader can be used to read the YAML file, with the resulting objects
supplied to the AgentGroup.
