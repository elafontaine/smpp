SMPP (a golang library)
=======================

[![codecov](https://codecov.io/gh/elafontaine/smpp/branch/master/graph/badge.svg?token=5A5N54FX17)](https://codecov.io/gh/elafontaine/smpp)

Why another smpp library?
-------------
Because I believe doing something using basic types is easier on people to understand. Having to use Specific types for
instantiating helps, but when you have to expect an object to be multiple types and handle special cases, I believe a
single type is easier to manage. Also because as soon as you go "full OOP", it becomes easy to believe people should
have the same mindset of the domain than you (which is often corrupted by the technical challenges to implement such
domain).

For example `gosmpp` is a full type-oriented approach which I was told is the golang idiomatic representation of SMPP. I
spent a while in there trying to see how to use it without going through many types. I had to make objects after objects
of certain specific types **(don't get me wrong, some people like that model, I'm just not able to think that way)**. As I was unsuccessful to make
a simple ESME send Submit_SM, I looked at my python implementation and decided it was simpler to just implement it in
go. There is also the added complexity of having multiple types in your head in that way of modeling the code, which is
something I know human are not good at doing (at most 2-3 types in their heads).

Hence, re-iterating what was said in the first paragraph, I believe people will better understand when we use a map as
this is the way human also use their brain to think. If you keep the types to the basic types, there is a lot of things
that will be useable out of the box.

Composition over inheritance
----------------------------

OOP was initially meant for `struct` and behaviours.  This is exactly why I like the 
golang paradigm ; you can have interfaces and define behaviours, but it will be 
attach to a `struct` which can be used from another composition.  This means we should 
use that same pattern for our own data.  This is why we can intantiate an ESME by passing
a server address and it will connect, but the struct is receiving a socket underneath.  This is done as an ESME 
isn't responsible for the TCP/TLS (transport) layer, it's responsible for the smpp layer.  So as long as you pass an ESME a socket object, we should be able to run an ESME over it.  Same for SMSC.  You can find an example of that in the `example` folder.


How to use the library ?
---------------

Every PDU exchanged will be of type `PDU`. That type contains a `Header` which always contain the SMPP headers values.
It also contains the `Body` which in turns contains a `mandatoryParameter` field for the mandatory SMPP parameter of the PDU command used. Finally, the `Body`
also contain the `optionalParameter` field which is a map of the optional parameters present on the pdu.

Although, being able to make a PDU object is nice, it's often an annoyance of having to instantiate all the mandatory parameters and their defaults.  For that matter, the `pdu_builder.go` file contains multiple builder for the SMPP PDU types and their defaults as well as the building functions to assign a value to a parameter (weither it's in the `Header` or `mandatoryParameter`,  the `optionalParameter` hasn't been made official yet in how we want to handle it).

So, if you want to be making your own PDUs
```
bind_pdu = NewBindTransmitter().WithSystemId(systemID).WithPassword(password).WithSequenceNumber(sequence_number)

pduBytes, err := EncodePdu(bind_pdu)
```

Now, this is not really useful by itself, unless you have an ESME object you could use to abstract much of the complexity away ; 

```
clientSocket, err := net.Dial("tcp", serverAddress.String())
e:= NewEsme(clientSocket)

```
The `ESME` object has some convenience functions at the moment and *but is currently underwork and it may changes*... 
I know that I want people to be able to ask for "one ESME, please!" and be able to provide the framework 
for handling the SMPP exchange protocol value through the golang channels.  So the ESME will probably evolve 
into a mechanics of passing PDU objects or bytes through a channel (probably bytes as there is no validation
on PDU objects themselves, and I want the user to receive the error, not the internals of the ESME).

The `SMSC` object is currently not made for production use and instantiate ESMEs for each connection.  There is some
logic at the moment for dispatching messages, but it would probably be using the same as the ESME when they're ready.

How to register custom functions for managing the SMPP session
--------------------------------------------------------------

Currently being worked on.


