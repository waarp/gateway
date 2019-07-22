Supprimer un compte
===================

.. http:delete:: /api/partners/(partner)/accounts/(account)

   Supprime le compte de l'utilisateur `account` rattaché au partenaire
   portant le nom `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          DELETE /api/partners/partenaire1/accounts/utilisateur1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: Le compte a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte ou le partenaire demandé n'existe pas

   :Example:
       .. code-block:: http

          HTTP/1.1 204 NO CONTENT