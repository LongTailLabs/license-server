Internet Restriction Monitor (IRateMONk)
========================================

Easy daemon that runs and tracks counts and applies restrictions to incoming signals for access.

Concepts
--------

* Consumer: This is an entity, a group of people or company. Whoever is asking for access.
* Application: The application which is requesting access or what application the access is being requsted for.
* Client: The specific person/user/workstation asking for access.

Endpoints
---------

To create a Consumer.

.. code-block:: shell
	
	http -v POST localhost:3000/consumers/ id=starbucks name="Dumb Starbucks"

.. code-block:: http

	POST /consumers/ HTTP/1.1
	Accept: application/json
	Accept-Encoding: gzip, deflate, compress
	Content-Length: 45
	Content-Type: application/json; charset=utf-8
	Host: localhost:3000
	User-Agent: HTTPie/0.7.2

	{
	    "id": "starbucks", 
	    "name": "Dumb Starbucks"
	}

	HTTP/1.1 200 OK
	Content-Length: 0
	Content-Type: text/plain; charset=utf-8
	Date: Mon, 24 Feb 2014 01:19:57 GMT

