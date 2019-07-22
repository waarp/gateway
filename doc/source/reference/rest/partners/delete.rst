Supprimer un partenaire
=======================

.. http:delete:: /api/partners/(partner)

   Supprime le partenaire portant le nom `partner`.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          DELETE /api/partners/partenaire1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 204: Le partenaire a été supprimé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :Example:
       .. code-block:: http

          HTTP/1.1 204 NO CONTENT