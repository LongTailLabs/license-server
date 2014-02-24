Internet Restriction Monitor (IRateMONk)
========================================

Easy daemon that runs and tracks counts and applies restrictions to incoming signals for access.

Concepts
--------

* Consumer: This is an entity, a group of people or company. Whoever is asking for access.
* Application: The application which is requesting access or what application the access is being requsted for.
* Client: The specific person/user/workstation asking for access.
* Restriction: This is a parameter which will limit the possibility of receiving persmission from the server to use the application.
* Signal: This is a request from the application for access. These requests are counted.

Endpoints
---------

.. note::

	httpie will be used for all of the documentation requests however I will not show the command every time I use it.

.. note:: 

	to save space the number of headers shown in the responses have been limited.

Consumers
---------

To create a Consumer.

.. code-block:: shell
	
	http -v POST localhost:3000/consumers/ id=starbucks name="Dumb Starbucks"

.. code-block:: http

	POST /consumers/ HTTP/1.1
	Content-Length: 45
	Content-Type: application/json; charset=utf-8

	{
	    "id": "starbucks", 
	    "name": "Dumb Starbucks"
	}

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 0
	Content-Type: text/plain; charset=utf-8

Requests are validated to some extend


.. code-block:: http

	POST /consumers/ HTTP/1.1
	Content-Length: 26
	Content-Type: application/json; charset=utf-8

	{
	    "name": "Dumb Starbucks"
	}

.. code-block:: http

	HTTP/1.1 422 status code 422
	Content-Length: 41
	Content-Type: application/json; charset=utf-8

	{
	    "fields": {
	        "Id": "Required"
	    }, 
	    "overall": {}
	}

You also cannot add the same Consumer twice.

.. code-block:: http

	POST /consumers/ HTTP/1.1
	Content-Length: 45
	Content-Type: application/json; charset=utf-8

	{
	    "id": "starbucks", 
	    "name": "Dumb Starbucks"
	}

.. code-block:: http

	HTTP/1.1 409 Conflict
	Content-Length: 79
	Content-Type: application/json; charset=UTF-8

	{
	    "Context": {
	        "id": "starbucks", 
	        "name": "Dumb Starbucks"
	    }, 
	    "Error": "Already exists"
	}

Listing consumers back is how you would expect.

.. code-block:: http

	GET /consumers/ HTTP/1.1

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 76
	Content-Type: application/json; charset=UTF-8

	[
	    {
	        "_id": "world", 
	        "name": "world"
	    }, 
	    {
	        "_id": "starbucks", 
	        "name": "Dumb Starbucks"
	    }
	]

And you can access a single instance by using the ``_id`` listed in the response.

.. code-block:: http
	
	GET /consumers/starbucks HTTP/1.1

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 43
	Content-Type: application/json; charset=UTF-8

	{
	    "_id": "starbucks", 
	    "name": "Dumb Starbucks"
	}

Applications
------------

Applications work in precisely the same way as consumers except you use the word `applications` wherever you would use the word consumer.

.. code-block:: http

	GET /applications HTTP/1.1

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 76
	Content-Type: application/json; charset=UTF-8

	[
	    {
	        "_id": "world", 
	        "name": "world"
	    }, 
	    {
	        "_id": "starbucks", 
	        "name": "Dumb Starbucks"
	    }
	]


Signalling
----------

Signalling is the process by which an application requests permission to proceed granting access to the client attempting to load protected or limited information.

The signalling endpoint is a POST request that follows this pattern:

``/signal/:consumer/:application/:action``

Consumer and application we've already covered. ``action`` is a bucket for grouping counts. For instance you may want to track accesses to the information as well as installs. These two actions could be defined in the system whether they have limits placed on them or not.

If the client is allowed access (covered in Restrictions) the action specified will be incremented accordingly.

Currently there are only two available actions:

* use
* access

**Client** keys are required. Whether everyone uses the same key or everyone has a unique id a json body with an ``id`` field must be submitted to the client endpoint for tracking.

Whether it is attached to a user, login, machine or company is up to how you want the limitations to be enforced.

.. code-block:: shell
	
	http -v POST localhost:3000/signal/world/hello/use id=aaf2730ee

.. code-block:: http

	POST /signal/world/hello/use HTTP/1.1
	Content-Length: 19
	Content-Type: application/json; charset=utf-8

	{
	    "id": "aaf2730ee"
	}

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 17
	Content-Type: application/json; charset=UTF-8

	{
	    "accepted": true
	}

It's possible to be denied based on the current restrictions in play.

.. code-block:: http
	
	POST /signal/world/hello/use HTTP/1.1
	Content-Length: 14
	Content-Type: application/json; charset=utf-8

	{
	    "id": "xvfg"
	}

.. code-block:: http
	
	HTTP/1.1 400 Bad Request
	Content-Length: 100
	Content-Type: application/json; charset=UTF-8

	{
	    "accepted": false, 
	    "errors": [
	        "The maximum was reached on the counter, 'use'. 9 meets or exceeds 4."
	    ]
	}


Restrictions
------------

A restriction is a class of behavior defined by a filter. Currently there are only two filters:

* maxCount
* netAddr

Here are the available restrictions endpoints.

``GET /restrictions/:consumer/:application``: List the restrictions on this consumer+application.

``POST /restrictions/:consumer/:application``: Add a new restriction (detailed below)

``PUT /restrictions/:consumer/:application``: Replace all restrictions with this restriction.

``DELETE /restrictions/:consumer/:application``: Delete a resitriction. The exact parameters used in the current restriction must be used in the ``DELETE`` body.

When submitting a new restriction the ``class`` key must be one of these two values or the system will reject the request.

**Max Count**

The way the system counts is fairly open. Any signal/counter can be asked to be incremented and restricted. As mentioned above the only two counters so far are:

* use
* access

Let's see an example of adding a ``maxCount`` restriction:

.. code-block:: shell

	$ http -v put localhost:3000/restrictions/world/hello class=maxCount counter=use maximum:=4

.. code-block:: http
	
	PUT /restrictions/world/hello HTTP/1.1
	Content-Length: 53
	Content-Type: application/json; charset=utf-8

	{
	    "class": "maxCount", 
	    "counter": "use", 
	    "maximum": 4
	}

.. code-block:: http

	HTTP/1.1 200 OK
	Content-Length: 0
	Content-Type: text/plain; charset=utf-8


**Network Address**

Well, this does what you think it does. It takes a CIDR network address and limits all requests to ones originating from that location. (Careful about NAT or Proxy services)

.. code-block:: shell

	$ http -v post localhost:3000/restrictions/world/hello class=netAddr cidr=127.0.0.0/8

.. code-block:: http

	POST /restrictions/world/hello HTTP/1.1
	Content-Length: 43
	Content-Type: application/json; charset=utf-8

	{
	    "cidr": "127.0.0.0/8", 
	    "class": "netAddr"
	}

.. code-block:: http
	
	HTTP/1.1 200 OK
	Content-Length: 0
	Content-Type: text/plain; charset=utf-8


Internally all restrictions are stored as a list of validators on the consumer and application.

.. code-block:: javascript

	{
	  "_id": ObjectId("530a9981b6cfc08f7b3e966f"),
	  "application": "hello",
	  "consumer": "world",
	  "restrictions": [
	    {
	      "class": "maxCount",
	      "counter": "use",
	      "maximum": 4
	    },
	    {
	      "class": "netAddr",
	      "cidr": "127.0.0.0/8"
	    }
	  ]
	}


Access Log
------------

This is implemented, no endpoint yet though.