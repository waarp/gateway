Ajouter un identifiant SMTP
===========================

.. http:post:: /api/email/credentials

   Ajoute un nouvel identifiant SMTP.

   :reqheader Authorization: Les identifiants de l'utilisateur REST.

   :reqjson string emailAddress: L'adresse email de l'expéditeur.
   :reqjson string serverAddress: L'adresse du serveur SMTP.
   :reqjson string login: Le login de connexion au serveur SMTP.
   :reqjson string password: Le mot de passe de connexion au serveur SMTP.

   :statuscode 201: L'identifiant SMTP a été créé avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resheader Location: Le chemin d'accès au nouvel identifiant SMTP créé

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/email/credentials HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 116

         {
           "emailAddress": "gateway@example.com",
           "serverAddress": "smtp.example.com:587",
           "login": "gateway",
           "password": "secret"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/email/credentials/gateway@example.com
