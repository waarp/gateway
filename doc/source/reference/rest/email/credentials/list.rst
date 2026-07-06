Lister les identifiants SMTP
============================

.. http:get:: /api/email/credentials

   Renvoie une liste des identifiants SMTP connus.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les identifiants seront triés *(défaut: email+)*
   :type sort: [email+\|email-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resjson array credentials: La liste des identifiants SMTP demandés
   :resjsonarr string emailAddress: L'adresse email de l'expéditeur.
   :resjsonarr string serverAddress: L'adresse du serveur SMTP.
   :resjsonarr string login: Le login de connexion au serveur SMTP.
   :resjsonarr string password: Le mot de passe de connexion au serveur SMTP.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/email/credentials?limit=10&sort=email+ HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 160

         {
           "credentials": [{
             "emailAddress": "gateway@example.com",
             "serverAddress": "smtp.example.com:587",
             "login": "gateway",
             "password": "secret"
           }]
         }
