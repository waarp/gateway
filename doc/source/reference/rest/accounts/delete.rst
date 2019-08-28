Supprimer un compte
===================

.. http:delete:: /api/accounts/(int:account_id)

   Supprime le compte de l'utilisateur portant le numéro ``account_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          DELETE /api/accounts/1234 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: Le compte a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :Example:
       .. code-block:: http

          HTTP/1.1 204 NO CONTENT