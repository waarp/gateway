Consulter un utilisateur
========================

.. http:get:: /api/users/(int:user_id)

   Renvoie l'utilisateur portant le numéro ``user_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET https://my_waarp_gateway.net/api/users/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: L'utilisateur a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   :resjson number id: L'identifiant unique de l'utilisateur
   :resjson string username: Le nom de l'utilisateur

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 41

          {
            "id": 1,
            "username": "toto"
          }