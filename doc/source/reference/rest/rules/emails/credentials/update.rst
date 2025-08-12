Modifier un identifiant SMTP existant
=====================================

.. http:patch:: /api/email/credentials/(string:email)

   Crée un nouvel identifiant SMTP.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 201: L'identifiant a été modifié avec succès.
   :statuscode 400: Requête invalide.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: L'identifiant demandé n'existe pas.

   :resqson string emailAddress: L'adresse email d'envoi.
   :resqson string serverAddress: L'adresse (port inclus) du serveur SMTP servant
     à envoyer les emails.
   :resqson string login: Le nom d'utilisateur à utiliser pour l'authentification SMTP.
   :resqson string password: Le mot de passe utilisateur.

   :resheader Location: Le chemin d'accès du nouvel identifiant créé

   **Exemple de requête**

   .. code-block:: http

      PATCH https://my_waarp_gateway.net/api/email/credentials/waarp@example.com HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 197

      {
        "emailAddress": "waarp_new@example.com",
        "serverAddress": "smtp.example.com:587",
        "login": "waarp_new",
        "password": "sesame"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/email/credentials/waarp_new@example.com
