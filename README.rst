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

And you can access a single instance by using the `_id` listed in the response.

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

