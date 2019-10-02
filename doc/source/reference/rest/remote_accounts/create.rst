Ajouter un compte partenaire
============================

.. http:post:: /api/remote_accounts

   Ajoute un nouveau compte partenaire avec les informations renseignées en format JSON dans
   le corps de la requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number remoteAgentID: L'identifiant unique du partenaire distant auquel
      le compte est rattaché
   :reqjson string login: Le login du compte
   :reqjson string password: Le mot de passe du compte

   **Exemple de requête**

       .. code-block:: http

          POST https://my_waarp_gateway.net/api/remote_accounts HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 109

          {
            "remoteAgentID": 1,
            "login": "toto",
            "password": "titi"
          }


   **Réponse**

   :statuscode 201: Le compte a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouveau compte créé

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/remote_accounts/1
