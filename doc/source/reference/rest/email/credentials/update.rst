Modifier un identifiant SMTP
============================

.. http:patch:: /api/email/credentials/(string:emailAddress)

   Met à jour l'identifiant SMTP demandé. Les champs omis resteront inchangés
   (mise à jour partielle).

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :reqjson string emailAddress: La nouvelle adresse email de l'expéditeur.
   :reqjson string serverAddress: La nouvelle adresse du serveur SMTP.
   :reqjson string login: Le nouveau login de connexion au serveur SMTP.
   :reqjson string password: Le nouveau mot de passe de connexion au serveur SMTP.

   :statuscode 201: L'identifiant SMTP a été mis à jour avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'identifiant SMTP demandé n'existe pas

   :resheader Location: Le chemin d'accès à l'identifiant SMTP mis à jour

   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/email/credentials/gateway@example.com HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 28

         {
           "password": "new-secret"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/email/credentials/gateway@example.com
