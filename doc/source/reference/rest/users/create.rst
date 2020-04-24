Ajouter un utilisateur
======================

.. http:post:: /api/users

   Ajoute un nouvel utilisateur avec les informations renseignées en format JSON
   dans le corps de la requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string username: Le nom de l'utilisateur
   :reqjson string password: Le mot de passe de l'utilisateur

   :statuscode 201: L'utilisateur a été créé avec succès
   :statuscode 400: Un ou plusieurs des paramètres de l'utilisateur sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resheader Location: Le chemin d'accès au nouvel utilisateur créé


   .. admonition:: Exemple de requête

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/users HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 87

         {
           "username": "toto",
           "password": "titi"
         }

   .. admonition:: Exemple de réponse

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/users/toto