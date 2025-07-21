Consulter un identifiant SMTP
=============================

.. http:get:: /api/email/credentials/(string:email)

   Renvoie l'identifiant SMTP demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 200: L'identifiant a été renvoyé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: L'identifiant demandé n'existe pas.

   :resjson string emailAddress: L'adresse email d'envoi.
   :resjson string serverAddress: L'adresse (port inclus) du serveur SMTP servant
     à envoyer les emails.
   :resjson string login: Le nom d'utilisateur à utiliser pour l'authentification SMTP.
   :resjson string password: Le mot de passe utilisateur.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/email/credentials/waarp@example.com HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 194

      {
        "emailAddress": "waarp@example.com",
        "serverAddress": "smtp.example.com:587",
        "login": "waarp",
        "password": "sesame"
      }
