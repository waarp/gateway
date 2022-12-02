Annuler plusieurs transferts
============================

.. http:delete:: /api/transfers

   Annule tous les transferts ayant le statut renseigné dans les paramètres de
   requête.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param target: Filtre les transferts à annuler suivant leur statut.
      Valeurs possibles : ``planned``, ``running``, ``paused``, ``interrupted``,
      ``error``, et ``all``.
   :type sort: string

   :statuscode 202: Les transferts ont été annulés avec succès
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   |

   **Exemple de requête**

      .. code-block:: http

         DELETE https://my_waarp_gateway.net/api/transfers?target=error HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
