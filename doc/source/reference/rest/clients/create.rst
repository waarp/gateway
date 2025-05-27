Créer un client
===============

.. http:post:: /api/clients/(string:client_name)

   Ajoute un nouveau client avec les informations renseignées en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom du client.
   :reqjson string protocole: Le protocole du client.
   :reqjson string address: L'adresse locale du client (en format [adresse:port]).
   :reqjson object protoConfig: La configuration du client encodé sous forme
      d'un objet JSON. Cet objet dépend du protocole.
   :reqjson number nbOfAttempts: Le nombre de fois qu'un transfert effectué avec
      ce client sera retenté automatiquement en cas d'échec.
   :reqjson number firstRetryDelay: Le délai (en secondes) entre la tentative
      originale d'un transfert et la première reprise automatique.
   :reqjson number retryIncrementFactor: Le facteur par lequel le délai ci-dessus
      est multiplié à chaque nouvelle tentative d'un transfert donné.

   :statuscode 201: Le client a été créé avec succès.
   :statuscode 400: Un ou plusieurs des paramètres du client sont invalides.
   :statuscode 401: Authentification d'utilisateur invalide.

   :resheader Location: Le chemin d'accès au client client créé.


   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/clients/ HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 176

         {
           "name": "sftp_client",
           "protocol": "sftp",
           "address": "0.0.0.0:2222",
           "protoConfig": {}
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/clients/sftp_client