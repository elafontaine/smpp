SMPP (a golang library)
=======================

Why another ?
-------------
Because I believe doing something using basic types is easier on people to understand.  Having to use Specific types for 
instantiating helps, but when you have to expect an object to be multiple types and handle special cases, I believe a 
single type is easier to manage.  Also because as soon as you go "full OOP", it becomes easy to believe people should 
have the same mindset of the domain than you (which is often corrupted by the technical challenges to implement such
domain).  

For example `gosmpp` is a full type-oriented approach which I was told is the golang idiomatic representation of SMPP.
I spent a while in there trying to see how to use it without going through many types.  I had to make objects after 
objects of certain specific types (don't get me wrong, some people like that model, I just don't).  As I was unsuccessful
to make a simple ESME send Submit_SM, I looked at my python implementation and decided it was simpler to just implement 
it in go.  There is also the added complexity of having multiple types in your head in that way of modeling the code,
which is something I know human are not good at doing (at most 2-3 types in their heads).

Hence, re-iterating what was said in the first paragraph, I believe people will better understand when we use a map as
this is the way human also use their brain to think.  If you keep the types to the basic types, there is a lot of things
that will be useable out of the box.


How to use it ?
---------------
For now, I only did the decrypting of a PDU packet.  The encrypting test will be using it to reconstruct the exact same 
PDU through an "decode -> encode" stream of tests.

Every PDU exchanged will be of type `PDU`.  That type contains a `Header` which always container the SMPP headers value.
It also contains the `mandatoryParameter` field for the mandatory SMPP parameter of the PDU command used.  Finally, it 
also contain the `optionalParameter` field which is a map of the optional parameters present on the pdu.




