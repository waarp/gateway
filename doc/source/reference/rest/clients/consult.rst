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
   :resjson array partners: La liste des partenaires rattachés au client. Voir
      :ref:`rest_partners_list` pour plus de détails sur la structure de cette liste.


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
        "partners": "partners": [{
          "name": "openssh",
          "address": "10.0.0.1:22",
          "protoConfig": {},
          "authorizedRules": {
            "sending": [],
            "reception": []
          },
          "accounts": [{
            "login": "titi",
            "authorizedRules": {
              "sending": [],
              "reception": []
            }
          }]
        }]
      }
