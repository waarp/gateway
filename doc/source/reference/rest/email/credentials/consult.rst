Consulter un identifiant SMTP
==============================

.. http:get:: /api/email/credentials/(string:emailAddress)

   Renvoie l'identifiant SMTP demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: L'identifiant SMTP a été renvoyé avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'identifiant SMTP demandé n'existe pas

   :resjson string emailAddress: L'adresse email de l'expéditeur.
   :resjson string serverAddress: L'adresse du serveur SMTP.
   :resjson string login: Le login de connexion au serveur SMTP.
   :resjson string password: Le mot de passe de connexion au serveur SMTP.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/email/credentials/gateway@example.com HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 116

         {
           "emailAddress": "gateway@example.com",
           "serverAddress": "smtp.example.com:587",
           "login": "gateway",
           "password": "secret"
         }
