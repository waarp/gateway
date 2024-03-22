Remplacer un client
===================

.. http:put:: /api/clients/(string:client_name)

   Remplace le partenaire demandé par celui renseigné en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du client.
   :reqjson string protocol: Le protocole du client.
   :reqjson string localAddress: L'adresse locale du client (en format [adresse:port])
   :reqjson object protoConfig: La configuration du client encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.

   :statuscode 201: Le client a été modifié avec succès
   :statuscode 400: Un ou plusieurs des paramètres du client sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le client demandé n'existe pas

   :resheader Location: Le chemin d'accès au client modifié.


   |

   **Exemple de requête**

      .. code-block:: http

         PUT https://my_waarp_gateway.net/api/clients/sftp_client HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 148

         {
           "name": "new_sftp_client",
           "protocol": "sftp",
           "localAddress": "0.0.0.0:2223",
           "protoConfig": {}
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/clients/new_sftp_client