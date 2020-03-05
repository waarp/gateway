Supprimer un utilisateur
========================

.. http:delete:: /api/users/(int:user_id)

   Supprime l'utilisateur portant le numéro ``user_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          DELETE https://my_waarp_gateway.net/api/users/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: L'utilisateur a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'utilisateur demandé n'existe pas

   :Example:
       .. code-block:: http

          HTTP/1.1 204 NO CONTENT