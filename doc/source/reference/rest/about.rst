Statut de Gateway
=================

.. http:get:: /api/about

    Renvoie les informations sur l'instance Gateway et sur le statut de ses
    services internes.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le service est actif
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Server: La version de l'instance Gateway.
   :resheader Date: L'heure UTC actuelle en format :rfc:`HTTP-Date <7231#section-7.1.1.1>`
      standard.
   :resheader Waarp-Gateway-Date: L'heure locale de Gateway en format :rfc:`1123`.

   :resjson array coreServices: La liste des services de base de Gateway.

      * ``name`` (*string*) - Le nom du service.
      * ``state`` (*string*) - L'état du service.
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur.

   :resjson object servers: La liste des serveurs de transfert de Gateway.

      * ``name`` (*string*) - Le nom du serveur.
      * ``state`` (*string*) - L'état du serveur
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur.

   :resjson object clients: La liste des clients de transfert de Gateway.

      * ``name`` (*string*) - Le nom du client.
      * ``state`` (*string*) - L'état du client
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur.

   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/status HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 212

      {
        "coreServices": [
            {
                "name": "Admin",
                "state": "Running",
                "reason": ""
            }, {
                "name": "Database",
                "state": "Error",
                "reason": "Exemple de message d'erreur"
            }, {
                "name": "Controller",
                "state": "Offline",
                "reason": ""
            }
        ],
        "servers": [
            {
                "name": "serveur_sftp",
                "state": "Running",
                "reason": ""
            }, {
                "name": "serveur_r66",
                "state": "Offline",
                "reason": ""
            }
        ],
        "clients": [
            {
                "name": "client_sftp",
                "state": "Error",
                "reason": "Autre exemple de message d'erreur"
            }, {
                "name": "client_r66",
                "state": "Running",
                "reason": ""
            }
        ]
      }
