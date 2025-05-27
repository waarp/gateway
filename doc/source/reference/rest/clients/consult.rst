Consulter un client
===================

.. http:get:: /api/clients/(string:client_name)

   Renvoie les informations du client portant le nom ``client_name``.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Les informations du client ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le client demandé n'existe pas

   :resjson string name: Le nom du client.
   :resjson string localAddress: L'adresse locale du client (en format [adresse:port]).
   :resjson string protocol: Le protocole utilisé par le client.
   :resjson object protoConfig: La configuration du client encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :resjson number nbOfAttempts: Le nombre de fois qu'un transfert effectué avec
      ce client sera retenté automatiquement en cas d'échec.
   :resjson number firstRetryDelay: Le délai (en secondes) entre la tentative
      originale d'un transfert et la première reprise automatique.
   :resjson number retryIncrementFactor: Le facteur par lequel le délai ci-dessus
      est multiplié à chaque nouvelle tentative d'un transfert donné.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/clients/sftp_client HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 271

      {
        "name": "sftp_client",
        "protocol": "sftp",
        "address": "0.0.0.0:2222",
        "protoConfig": {},
        "nbOfAttempts": 5,
        "firstRetryDelay": 90,
        "retryIncrementFactor": 1.5
      }
