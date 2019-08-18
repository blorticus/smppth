# smpp-test-harness

## Overview

This repository contains a series of tools for testing SMPP 3.4 architectures, as
well as methods to simplify the construction of custom SMPP test tools.  It
is written in golang and depends on the smpp-go library.  On top of that library,
it adds a simplistic ESME (client) and SMSC (server) implementation, as well
as a method for launching and controlling multiple ESME and/or SMSC instances.
Furthermore, it adds a terminal-based UI for interacting with the launched
ESME and SMSC applications.

## Libraries

smpp-go provides methods for encoding and decoding SMPP PDUs.  smpp-test-harness
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

## Built-in Test Harness

There is a built-in test harness in `app/`, called `smpp-test-harness`.  It is started
as follows:

```bash
smpp-test-harness run esmes|smscs /path/to/config.yaml
```

An instance of `smpp-test-harness` run a set of ESME Agents or a set of SMSC Agents.
The Agents are described in the `config.yaml` file, and if the harness is running
ESME Agents, the `config.yaml` also describes the peers to which each ESME will bind.
The test harness only performs transceiver binds.  The format of `config.yaml` is
described below.

