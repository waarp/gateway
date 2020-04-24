Modifier un compte local
========================

.. http:put:: /api/servers/(string:server_name)/accounts/(string:login)

   Met à jour le compte donné avec les informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string login: Le login du compte
   :reqjson string password: Le mot de passe du compte

   :statuscode 201: Le compte a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resheader Location: Le chemin d'accès au compte modifié


   .. admonition:: Exemple de requête

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/servers/sftp_server/accounts/toto HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 96

         {
           "login": "toto_new",
           "password": "titi_new"
         }

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/servers/sftp_server/accounts/toto_new